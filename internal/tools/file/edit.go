package filetools

import (
	"fmt"
	"os"
	"strings"

	"github.com/Gabriel-Araujo/network_agent/pkg/util"
)

// EditFile troca a única ocorrência de oldText por newText no arquivo em
// filePath, resolvido contra workingDirectory. Devolve o que o modelo lê, as
// linhas que a edição removeu (só no sucesso; ver removedLines) e a falha
// tipada (ver ReadFile).
func EditFile(workingDirectory string, filePath string, oldText string, newText string) (string, []string, error) {
	path, err := util.SafePath(workingDirectory, filePath)

	if err != nil {
		log.Error("caminho inválido", "path", filePath, "err", err)
		return fmt.Sprintf("Error %s", err), nil, fmt.Errorf("%w: %w", ErrOutsideRoot, err)
	}

	stats, err := os.Stat(path)
	if err != nil {
		return "Error: File not found.", nil, ErrNotFound
	}

	if stats.Size() > int64(MAX_EDIT_FILE_SIZE) {
		return fmt.Sprintf("Error: File is too large. File has %s bu the limit is %s", util.FormatFileSize(stats.Size()), util.FormatFileSize(int64(MAX_EDIT_FILE_SIZE))), nil, ErrTooLarge
	}

	content, err := os.ReadFile(path)
	if err != nil {
		log.Error("falha ao ler arquivo", "path", path, "err", err)
		return "Error: failed to read File content", nil, fmt.Errorf("lendo o arquivo para editar: %w", err)
	}

	contentStr := string(content)
	switch occurrences := strings.Count(contentStr, oldText); {
	case occurrences == 0:
		return "Error: old_text not found in file", nil, ErrTextNotFound
	case occurrences > 1:
		return fmt.Sprintf("%s%d", EDIT_MULTIPLE_MATCHES_NOTICE, occurrences), nil, fmt.Errorf("%w: %d ocorrências", ErrTextNotUnique, occurrences)
	}

	statsAfterRead, err := os.Stat(path)
	if err != nil {
		log.Error("falha ao checar modificação do arquivo", "path", path, "err", err)
		return "Error: failed to check file staleness", nil, fmt.Errorf("checando a modificação do arquivo: %w", err)
	}

	if !statsAfterRead.ModTime().Equal(stats.ModTime()) {
		return "Error: File was modified by another process since it was last accessed.", nil, ErrStale
	}

	// A comparação é por linha inteira do arquivo, não pelo trecho: um old_text
	// que começa ou termina no meio de uma linha muda a linha toda, e o Aviso
	// tem de mostrá-la como estava.
	at := strings.Index(contentStr, oldText)
	head := contentStr[strings.LastIndex(contentStr[:at], "\n")+1 : at]
	tail, _, _ := strings.Cut(contentStr[at+len(oldText):], "\n")
	removed := removedLines(head+oldText+tail, head+newText+tail)

	newContentStr := contentStr[:at] + newText + contentStr[at+len(oldText):]
	err = os.WriteFile(path, []byte(newContentStr), 0644)
	if err != nil {
		log.Error("falha ao escrever arquivo", "path", path, "err", err)
		return "Error: failed to write File content", nil, fmt.Errorf("gravando o arquivo editado: %w", err)
	}

	if len(removed) > 0 {
		return "Success: File edited. Removed lines: " + strings.Join(removed, " | "), removed, nil
	}
	return "Success: File edited", nil, nil
}

// removedLines lista, na ordem de before, as linhas que after tem menos vezes
// (decisão do ticket 08 do mapa frr-file-builder, regra 1): comparadas sem
// espaços nas pontas, então reindentar e reordenar não é remover. Linha vazia
// e "!" sozinho não contam — são respiro da config, não config —, mas
// "! WARNING:" e todo comentário com texto contam.
func removedLines(before, after string) []string {
	kept := map[string]int{}
	for line := range strings.SplitSeq(after, "\n") {
		kept[strings.TrimSpace(line)]++
	}
	var removed []string
	for line := range strings.SplitSeq(before, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "" || line == "!":
		case kept[line] > 0:
			kept[line]--
		default:
			removed = append(removed, line)
		}
	}
	return removed
}
