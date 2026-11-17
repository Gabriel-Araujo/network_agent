package turn

import (
	"fmt"
	"slices"

	"github.com/Gabriel-Araujo/network_agent/internal/tools"
	filetools "github.com/Gabriel-Araujo/network_agent/internal/tools/file"
)

// O AVISO DA FALHA DE TOOL
//
// Uma tool que falhou é degradação do turno, não um turno que morreu: ela entra
// em Outcome.Warnings, o mesmo campo dos fallbacks da composição, e daí
// segue pelo caminho que já existe — a Message, a tira da bolha, o log e o
// cmd/cli (ticket 19, §1).
//
// O Aviso diz **o que o app fez**, e é por isso que ele existe: a recusa já
// chegava ao usuário pela boca do modelo, e o caso medido foi o modelo acusando
// o caminho do usuário quando o errado era a raiz. Um Aviso é a afirmação do
// app sobre a ação dele.

// acoes nomeia, com a preposição, o que cada file tool estava fazendo quando
// falhou: é a "ação" do Aviso. A preposição vem no mapa porque a busca recebe
// uma pasta ("busca em") e as outras três um arquivo ("leitura de").
//
// Uma tool fora do mapa — skill, sub-agente, argumentos ilegíveis — não tem
// caminho nem raiz (ticket 18, §2), e aparece pelo nome.
var acoes = map[string]string{
	filetools.READ_TOOL_NAME:   "leitura de",
	filetools.WRITE_TOOL_NAME:  "gravação de",
	filetools.EDIT_TOOL_NAME:   "edição de",
	filetools.SEARCH_TOOL_NAME: "busca em",
}

// warnFailures drena as falhas de tool de uma chamada de Chat para
// out.Warnings, deduplicadas por texto idêntico: duas tentativas no mesmo
// arquivo pelo mesmo motivo são uma entrada (ticket 19, §2).
//
// A comparação é contra a lista inteira, e não contra as falhas desta chamada,
// porque o turno tem duas chamadas de Chat — a final e o re-ask do diagrama —
// e a mesma recusa pode voltar na segunda.
func warnFailures(out *Outcome, failures []tools.Failure) {
	for _, fail := range failures {
		if aviso := failureWarning(fail); !slices.Contains(out.Warnings, aviso) {
			out.Warnings = append(out.Warnings, aviso)
		}
	}
}

// failureWarning escreve o Aviso de uma falha: ação + caminho **como o modelo
// pediu** + a pasta contra a qual ele resolveu + motivo curto (ticket 19, §3).
//
// A pasta, e não o absoluto por arquivo: ela é o knob que decide onde a tool
// olhou — o que o ticket 02 tornou efetivo e o 10 desdobrou em duas raízes —, e
// o absoluto já está no log.Error que a gaveta mostra. Ela falta em um caso só,
// o de nenhuma pasta escolhida, que é justamente o que não tem pasta.
//
// Sem a cláusula de continuação dos quatro avisos da composição ("continuing
// without..."): na tira, a degradação é a própria tira.
func failureWarning(fail tools.Failure) string {
	motivo := filetools.Reason(fail.Err)
	acao, ok := acoes[fail.Tool]
	if !ok {
		return fmt.Sprintf("a tool %s falhou: %s", fail.Tool, motivo)
	}
	if fail.Root == "" {
		return fmt.Sprintf(`%s "%s" falhou: %s`, acao, fail.Path, motivo)
	}
	return fmt.Sprintf(`%s "%s" falhou na pasta "%s": %s`, acao, fail.Path, fail.Root, motivo)
}
