// Package classifier classifica um input escolhendo, para cada Lista passada
// na chamada, exatamente um dos valores dela — o valor canônico — ou "" quando
// nada se encaixa. O contrato completo está em
// .agent/classifier/issues/02-contrato-do-classificador.md.
package classifier

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	llm "github.com/Gabriel-Araujo/network_agent/internal/agent"
)

// Valor é um valor candidato de uma Lista. Nome é o valor canônico: é o único
// que pode sair na classificação, sempre com esta grafia. Descricao é opcional
// e existe para o modelo desempatar valores vizinhos.
type Valor struct {
	Nome      string
	Descricao string
}

// Lista é um conjunto fechado de valores para uma dimensão, passado por quem
// chama. A classificação devolve um valor por lista — o Nome canônico de um
// dos Valores — ou "" quando nada se encaixa.
type Lista struct {
	Nome    string
	Valores []Valor
}

// maxTentativas é 1 chamada + 2 retries de correção: saída inválida é refeita
// no máximo 2 vezes com o erro colado na tentativa seguinte e, esgotadas as
// tentativas, o erro sobe explícito — nunca um valor inventado vaza.
const maxTentativas = 3

// completarFunc devolve o conteúdo cru da resposta do provedor. É a costura de
// teste: os testes injetam respostas canned, sem rede.
type completarFunc func(ctx context.Context, msgs []mensagem, schema map[string]any) (string, error)

// mensagem é um turno do prompt, já na forma que o provedor recebe.
type mensagem struct {
	Papel    string // "system" | "user" | "assistant"
	Conteudo string
}

// Do classifica input: um valor canônico por lista, ou "" quando nada se
// encaixa. modoRAG é o Modo RAG do classificador — flag como parâmetro da
// chamada, quem chama decide o modo.
func Do(ctx context.Context, input string, listas []Lista, modoRAG bool, agent *llm.Agent) (map[string]string, error) {
	if agent == nil {
		return nil, errors.New("classificador: agente nil")
	}
	return classificar(ctx, input, listas, modoRAG, completarDoAgent(agent))
}

// classificar roda o ciclo completo: validar as listas, montar o prompt e o
// schema, chamar o modelo e validar a resposta. A garantia de saída é a
// validação aqui — o provedor pode ignorar o schema em silêncio (ver
// .agent/classifier/assets/pesquisa-saida-estruturada.md), por isso 100% dos
// caminhos passam por pertencimento à lista.
func classificar(ctx context.Context, input string, listas []Lista, modoRAG bool, completar completarFunc) (map[string]string, error) {
	indice, err := validarListas(listas)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(input) == "" {
		return nil, errors.New("classificador: input vazio")
	}

	schema := montarSchema(listas)
	msgs := []mensagem{
		{Papel: "system", Conteudo: montarPrompt(listas, modoRAG)},
		{Papel: "user", Conteudo: input},
	}

	var ultimoErro error
	for tentativa := 1; tentativa <= maxTentativas; tentativa++ {
		bruto, err := completar(ctx, msgs, schema)
		if err != nil {
			ultimoErro = fmt.Errorf("chamada ao modelo (tentativa %d): %w", tentativa, err)
			continue
		}
		valores, err := validarResposta(bruto, listas, indice)
		if err == nil {
			return valores, nil
		}
		ultimoErro = fmt.Errorf("resposta rejeitada (tentativa %d): %w", tentativa, err)
		msgs = append(msgs,
			mensagem{Papel: "assistant", Conteudo: bruto},
			mensagem{Papel: "user", Conteudo: montarCorrecao(err)},
		)
	}
	return nil, fmt.Errorf("classificador: não classificou em %d tentativas: %w", maxTentativas, ultimoErro)
}

// indiceDeLista resolve o valor canônico de uma lista pela grafia normalizada:
// o modelo pode errar caixa/anotação, mas quem sai é a grafia da lista.
type indiceDeLista map[string]string // normalizado -> Nome canônico

// validarListas valida a entrada da chamada antes de qualquer ida ao modelo e
// monta o índice de resolução. Valores equivalentes na mesma lista ("BGP" e
// "bgp") são recusados: tornariam a resolução ambígua.
func validarListas(listas []Lista) (map[string]indiceDeLista, error) {
	if len(listas) == 0 {
		return nil, errors.New("classificador: nenhuma lista passada")
	}
	indice := make(map[string]indiceDeLista, len(listas))
	for _, l := range listas {
		nome := strings.TrimSpace(l.Nome)
		if nome == "" {
			return nil, errors.New("classificador: lista sem nome")
		}
		if _, repetida := indice[nome]; repetida {
			return nil, fmt.Errorf("classificador: lista %q repetida", nome)
		}
		if len(l.Valores) == 0 {
			return nil, fmt.Errorf("classificador: lista %q sem valores", nome)
		}
		porNormalizado := make(indiceDeLista, len(l.Valores))
		for _, v := range l.Valores {
			if strings.TrimSpace(v.Nome) == "" {
				return nil, fmt.Errorf("classificador: lista %q tem valor sem nome", nome)
			}
			n := normalizar(v.Nome)
			if anterior, equivalente := porNormalizado[n]; equivalente {
				return nil, fmt.Errorf("classificador: lista %q tem valores equivalentes %q e %q", nome, anterior, v.Nome)
			}
			porNormalizado[n] = v.Nome
		}
		indice[nome] = porNormalizado
	}
	return indice, nil
}

// validarResposta extrai o JSON da resposta e resolve cada lista. O que sai é
// sempre o valor canônico da lista (ou "") — a grafia do modelo nunca é
// repassada. Lista ausente, valor não-texto e valor fora da lista são erro.
func validarResposta(bruto string, listas []Lista, indice map[string]indiceDeLista) (map[string]string, error) {
	obj, err := extrairObjeto(bruto)
	if err != nil {
		return nil, err
	}
	saida := make(map[string]string, len(listas))
	for _, l := range listas {
		nome := strings.TrimSpace(l.Nome)
		cru, presente := obj[nome]
		if !presente {
			return nil, fmt.Errorf("lista %q ausente na resposta", nome)
		}
		var texto string
		if err := json.Unmarshal(cru, &texto); err != nil {
			return nil, fmt.Errorf("valor da lista %q não é texto: %v", nome, err)
		}
		if strings.TrimSpace(texto) == "" {
			saida[nome] = "" // vazio explícito: nada se encaixa
			continue
		}
		canonico, ok := indice[nome][normalizar(texto)]
		if !ok {
			return nil, fmt.Errorf("valor %q fora da lista %q (permitidos: %s)",
				texto, nome, strings.Join(nomesDosValores(l), ", "))
		}
		saida[nome] = canonico
	}
	return saida, nil
}

func nomesDosValores(l Lista) []string {
	nomes := make([]string, 0, len(l.Valores))
	for _, v := range l.Valores {
		nomes = append(nomes, v.Nome)
	}
	return nomes
}

var (
	fenceRe    = regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)```")
	anotacaoRe = regexp.MustCompile(`\s*\([^)]*\)\s*$`)
)

// extrairObjeto tolera code fences e texto ao redor — modelos locais emitem os
// dois mesmo instruídos a não fazer — mas o JSON precisa decodificar como
// objeto; fora isso, é erro.
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

// normalizar casa a grafia do modelo ao valor canônico: caixa, espaço e
// anotações entre parênteses no fim ("ospf (primary)") não mudam o valor — mas
// o que sai é sempre a grafia canônica da lista, nunca a do modelo.
func normalizar(s string) string {
	s = anotacaoRe.ReplaceAllString(strings.TrimSpace(s), "")
	return strings.ToLower(strings.TrimSpace(s))
}
