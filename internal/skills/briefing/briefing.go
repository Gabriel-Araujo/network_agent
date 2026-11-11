// Package briefing gera o briefing rico de uma pergunta de redes: devices,
// conexões, resumo do problema e pipeline de execução. É a skill preservada
// do antigo intent-analyser — rica, porém fora do fluxo do CLI. Contrato:
// .agent/classifier/issues/04-desenho-da-skill-do-briefing-rico.md.
package briefing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	llm "github.com/Gabriel-Araujo/network_agent/internal/agent"
	"github.com/Gabriel-Araujo/network_agent/internal/config"
	"github.com/Gabriel-Araujo/network_agent/internal/logger"
	"github.com/Gabriel-Araujo/network_agent/pkg/util"
)

var log = logger.Named("BRIEFING")

// ErrEspelhoNaoGravado sinaliza a falha de gravação do espelho do briefing: o
// briefing volta completo em memória e o path vem vazio — o
// espelho nunca derruba o caminho (checável com errors.Is).
var ErrEspelhoNaoGravado = errors.New("espelho do briefing não gravado")

// Device é um device do cenário — inclusive peers fora do lado FRR.
type Device struct {
	Label               string   `json:"label"`
	Role                string   `json:"role"`
	Platform            string   `json:"platform"`
	DaemonsAndProtocols []string `json:"daemonsAndProtocols"`
}

// Connection é um link/sessão entre dois devices do cenário.
type Connection struct {
	From               string `json:"from"`
	To                 string `json:"to"`
	InterfaceAndLink   string `json:"InterfaceAndLink"`
	ProtocolAndSession string `json:"protocolAndSession"`
}

// Briefing é a saída da skill: só as partes ricas, com os nomes e shapes de
// hoje. Os nomes das chaves são os do template do prompt — a validação e o
// espelho dependem deles.
type Briefing struct {
	Devices               []Device          `json:"devices"`
	Connections           []Connection      `json:"connections"`
	ProblemAndGoalSummary string            `json:"problemAndGoalSummary"`
	ExecutionPipeline     map[string]string `json:"executionPipeline"`
}

// maxTentativas é 1 chamada + 2 retries de correção: saída inválida é refeita
// no máximo 2 vezes com o erro colado na tentativa seguinte e, esgotadas as
// tentativas, o erro sobe explícito — sem fallback silencioso.
const maxTentativas = 3

// espelhoDir é o diretório do espelho, relativo ao diretório de dados do app
// (config.Dir()) e nunca ao diretório de trabalho. A nominação
// <slug>-<hash8>.json vive em pkg/util (compartilhada com o espelho do
// retrieval).
const espelhoDir = "tmp"

// completarFunc devolve o conteúdo cru da resposta do provedor. É a costura de
// teste: os testes injetam respostas canned, sem rede.
type completarFunc func(ctx context.Context, msgs []mensagem) (string, error)

// mensagem é um turno do prompt, já na forma que o provedor recebe.
type mensagem struct {
	Papel    string // "system" | "user" | "assistant"
	Conteudo string
}

// Do gera o briefing rico de input: o valor em memória, o path do espelho
// <config.Dir()>/tmp/<slug>-<hash8>.json e o erro — sentinela ErrEspelhoNaoGravado
// quando só o espelho falhou (briefing completo e path vazio nesse caso).
func Do(ctx context.Context, input string, agent *llm.Agent) (Briefing, string, error) {
	if agent == nil {
		return Briefing{}, "", errors.New("briefing: agente nil")
	}
	return do(ctx, input, completarDoAgent(agent))
}

// do roda o ciclo e grava o espelho. O arquivo é o JSON da mesma struct —
// espelho, não o produto: falha de gravação sobe como ErrEspelhoNaoGravado
// sem derrubar o valor em memória.
func do(ctx context.Context, input string, completar completarFunc) (Briefing, string, error) {
	b, err := analisar(ctx, input, completar)
	if err != nil {
		return Briefing{}, "", err
	}
	path, err := salvarEspelho(input, b)
	if err != nil {
		log.Warn("espelho do briefing não gravado — briefing devolvido em memória", "err", err)
		return b, "", fmt.Errorf("%w: %w", ErrEspelhoNaoGravado, err)
	}
	return b, path, nil
}

// analisar monta o prompt, chama o modelo e valida a resposta. A garantia de
// saída é a validação aqui — o provedor pode ignorar instruções de formato em
// silêncio (ver .agent/classifier/assets/pesquisa-saida-estruturada.md), por
// isso 100% dos caminhos passam pela validação no cliente.
func analisar(ctx context.Context, input string, completar completarFunc) (Briefing, error) {
	if strings.TrimSpace(input) == "" {
		return Briefing{}, errors.New("briefing: input vazio")
	}
	msgs := []mensagem{
		{Papel: "system", Conteudo: systemPrompt + "\n\n" + skillPrompt},
		{Papel: "user", Conteudo: input},
	}

	var ultimoErro error
	for tentativa := 1; tentativa <= maxTentativas; tentativa++ {
		bruto, err := completar(ctx, msgs)
		if err != nil {
			ultimoErro = fmt.Errorf("chamada ao modelo (tentativa %d): %w", tentativa, err)
			continue
		}
		b, err := validarResposta(bruto)
		if err == nil {
			return b, nil
		}
		ultimoErro = fmt.Errorf("resposta rejeitada (tentativa %d): %w", tentativa, err)
		msgs = append(msgs,
			mensagem{Papel: "assistant", Conteudo: bruto},
			mensagem{Papel: "user", Conteudo: montarCorrecao(err)},
		)
	}
	return Briefing{}, fmt.Errorf("briefing: não gerou briefing em %d tentativas: %w", maxTentativas, ultimoErro)
}

// chavesObrigatorias são as quatro partes do briefing: presentes em toda
// resposta, com o tipo decidido — nunca null, nunca inventadas.
var chavesObrigatorias = []string{"devices", "connections", "problemAndGoalSummary", "executionPipeline"}

// validarResposta extrai o JSON da resposta e decodifica as quatro partes nos
// tipos decididos. Chave ausente, null ou de tipo errado são erro — o modelo
// nunca dita a forma, o contrato dita.
func validarResposta(bruto string) (Briefing, error) {
	obj, err := extrairObjeto(bruto)
	if err != nil {
		return Briefing{}, err
	}
	for _, chave := range chavesObrigatorias {
		raw, presente := obj[chave]
		if !presente || strings.TrimSpace(string(raw)) == "null" {
			return Briefing{}, fmt.Errorf("chave %q ausente na resposta", chave)
		}
	}

	var b Briefing
	if err := json.Unmarshal(obj["devices"], &b.Devices); err != nil {
		return Briefing{}, fmt.Errorf("devices com tipo inválido: %w", err)
	}
	if err := json.Unmarshal(obj["connections"], &b.Connections); err != nil {
		return Briefing{}, fmt.Errorf("connections com tipo inválido: %w", err)
	}
	if err := json.Unmarshal(obj["problemAndGoalSummary"], &b.ProblemAndGoalSummary); err != nil {
		return Briefing{}, fmt.Errorf("problemAndGoalSummary com tipo inválido: %w", err)
	}
	if err := json.Unmarshal(obj["executionPipeline"], &b.ExecutionPipeline); err != nil {
		return Briefing{}, fmt.Errorf("executionPipeline com tipo inválido: %w", err)
	}
	return b, nil
}

var (
	fenceRe    = regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)```")
	corChaveRe = regexp.MustCompile(`^\s*"([^"]+)"`)
)

// extrairObjeto tolera code fences e texto ao redor — modelos locais emitem os
// dois mesmo instruídos a não fazer (mesma tolerância do Classificador) — mas
// o JSON precisa decodificar como objeto; fora isso, é erro.
func extrairObjeto(bruto string) (map[string]json.RawMessage, error) {
	t := strings.TrimSpace(bruto)
	candidatos := []string{t}
	for _, m := range fenceRe.FindAllStringSubmatch(t, -1) {
		candidatos = append(candidatos, m[1])
	}
	if i := strings.Index(t, "{"); i >= 0 {
		if j := strings.LastIndex(t, "}"); j > i {
			candidatos = append(candidatos, t[i:j+1])
		}
	}
	for _, c := range candidatos {
		var obj map[string]json.RawMessage
		if err := json.Unmarshal([]byte(strings.TrimSpace(c)), &obj); err == nil && obj != nil {
			return obj, nil
		}
	}
	return nil, errors.New("sem JSON objeto na resposta do modelo")
}

// salvarEspelho grava o JSON do briefing em espelhoDir com o nome
// <slug>-<hash8>.json derivado do input — o mesmo input regrava o mesmo
// espelho (sobrescreve, sem histórico de execuções). Retorna o caminho do
// arquivo criado.
//
// Sem SafePath, pela mesma razão do espelho do retrieval: o nome vem do
// util.Slugify, que não deixa separador, ponto nem ".." no caminho.
func salvarEspelho(input string, b Briefing) (string, error) {
	name := util.Slugify(input) + "-" + util.Hash8(input) + ".json"

	path := filepath.Join(config.Dir(), espelhoDir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return "", fmt.Errorf("serializando briefing: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("gravando %s: %w", path, err)
	}
	return path, nil
}
