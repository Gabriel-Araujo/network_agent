package classifier

import (
	"fmt"
	"strings"
)

// systemPrompt é a identidade do classificador: ele não conversa, ele escolhe.
const systemPrompt = `# Identity
You are a text classifier for a network engineering assistant (FRRouting). You do not chat: you read the user's request and choose values.

# Rules
- For EACH list below, choose EXACTLY ONE of its allowed values, spelled exactly as listed.
- If nothing in a list fits the request, use "" (empty string). "Nothing fits" is a valid answer; an invented value is never acceptable.
- Reply with a single JSON object only: one property per list and nothing else. No preamble, no code fences.`

// modoRAGPrompt liga o Modo RAG do classificador: a flag é parâmetro da
// chamada (quem chama decide o modo) e muda o comportamento interno — a saída
// alimenta os filtros de busca do RAG (Como o modo RAG alimenta a busca).
const modoRAGPrompt = `
# Modo RAG
This classification feeds RAG search filters. Prefer the specific value over "" whenever the request touches that protocol or daemon at all: the value scopes the document search for the whole request.`

// montarPrompt monta o prompt de sistema com as listas e seus valores — a
// descrição por valor é o que dá ao modelo material para desempatar.
func montarPrompt(listas []Lista, modoRAG bool) string {
	var b strings.Builder
	b.WriteString(systemPrompt)
	if modoRAG {
		b.WriteString(modoRAGPrompt)
	}
	b.WriteString("\n\n# Lists\n")
	for _, l := range listas {
		fmt.Fprintf(&b, "\n## %s\n", strings.TrimSpace(l.Nome))
		for _, v := range l.Valores {
			fmt.Fprintf(&b, "- %q", v.Nome)
			if d := strings.TrimSpace(v.Descricao); d != "" {
				fmt.Fprintf(&b, ": %s", d)
			}
			b.WriteString("\n")
		}
	}
	return b.String()
}

// montarCorrecao cola o erro da tentativa rejeitada na próxima chamada.
func montarCorrecao(err error) string {
	return fmt.Sprintf(`Your previous answer was rejected: %v

Answer again with a single JSON object only, one property per list, each value spelled exactly as one of that list's allowed values — or "" when nothing fits. Never invent a value.`, err)
}

// montarSchema pede ao provedor o formato de saída: um objeto com uma
// propriedade por lista, cada uma string com o enum dos valores permitidos mais
// o vazio explícito. É o mesmo contrato que a validação no cliente reforça — o
// provedor pode ignorar o schema em silêncio (ver
// .agent/classifier/assets/pesquisa-saida-estruturada.md), por isso a garantia
// é a validação, não o schema.
func montarSchema(listas []Lista) map[string]any {
	props := make(map[string]any, len(listas))
	required := make([]string, 0, len(listas))
	for _, l := range listas {
		nome := strings.TrimSpace(l.Nome)
		enum := []string{""}
		for _, v := range l.Valores {
			enum = append(enum, v.Nome)
		}
		props[nome] = map[string]any{"type": "string", "enum": enum}
		required = append(required, nome)
	}
	return map[string]any{
		"type":                 "object",
		"properties":           props,
		"required":             required,
		"additionalProperties": false,
	}
}
