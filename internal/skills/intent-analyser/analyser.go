package intentanalyser

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	llm "github.com/Gabriel-Araujo/network_agent/internal/llm/agent"
	"github.com/Gabriel-Araujo/network_agent/pkg/util"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

// reportDir é o diretório onde as análises de intenção são salvas,
// relativo ao diretório de trabalho atual.
const reportDir = ".agent/tmp"

// Do analisa a intenção de uma pergunta de redes, transformando-a num
// briefing estruturado em JSON, salva o resultado em disco e retorna
// o caminho absoluto do arquivo gerado.
func Do(ctx context.Context, input string, agent *llm.Agent) (string, error) {
	// O template de chat do modelo só aceita mensagem de sistema na primeira
	// posição. Instructions já vira uma, então o skillPrompt vai junto dela em
	// vez de virar um segundo system no meio do input (e o timestamp vai no
	// turno do usuário, em vez de num turno de assistant antes dele).
	response, err := agent.Client.Responses.New(ctx, responses.ResponseNewParams{
		Model:        agent.ModelName,
		Instructions: openai.String(systemPrompt + "\n\n" + skillPrompt),
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: responses.ResponseInputParam{
				responses.ResponseInputItemParamOfMessage(
					"Current timestamp="+time.Now().Format(time.RFC3339)+"\n\n"+input,
					responses.EasyInputMessageRoleUser,
				),
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("analisando intenção: %w", err)
	}

	path, err := saveReport(input, response.OutputText())
	if err != nil {
		return "", fmt.Errorf("salvando análise de intenção: %w", err)
	}

	return path, nil
}

// saveReport grava o conteúdo da análise em reportDir seguindo a convenção
// de nome <slug>-<hash8>.md e retorna o caminho do arquivo criado.
func saveReport(query, content string) (string, error) {
	name := Slugify(query) + "-" + Hash8(query) + ".json"

	path, err := util.SafePath(".", reportDir)
	if err != nil {
		return "", err
	}

	path = filepath.Join(path, name)
	dir := filepath.Dir(path)
	if err = os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", err
	}

	return path, nil
}

// Slugify converte uma query em um slug minúsculo, sem acentos, limitado a
// no máximo 5 palavras (separadas por hífen), contendo apenas [a-z0-9-].
func Slugify(input string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(input) {
		switch r {
		case 'á', 'à', 'â', 'ä', 'ã':
			r = 'a'
		case 'é', 'è', 'ê', 'ë':
			r = 'e'
		case 'í', 'ì', 'î', 'ï':
			r = 'i'
		case 'ó', 'ò', 'ô', 'ö', 'õ':
			r = 'o'
		case 'ú', 'ù', 'û', 'ü':
			r = 'u'
		case 'ç':
			r = 'c'
		}
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}

	parts := strings.FieldsFunc(b.String(), func(r rune) bool { return r == '-' })
	if len(parts) > 5 {
		parts = parts[:5]
	}
	return strings.Join(parts, "-")
}

// Hash8 retorna os primeiros 8 caracteres hexadecimais do SHA1 da query
// concatenada com o timestamp atual, garantindo nome de arquivo único.
func Hash8(input string) string {
	h := sha1.Sum([]byte(input + time.Now().String()))
	return hex.EncodeToString(h[:])[:8]
}
