package filetools

import (
	"fmt"
	"os"
	"strings"

	"github.com/Gabriel-Araujo/network_agent/pkg/util"
)

// EditFile troca a única ocorrência de oldText por newText no arquivo em
// filePath, resolvido contra workingDirectory. Devolve o que o modelo lê e a
// falha tipada (ver ReadFile).
func EditFile(workingDirectory string, filePath string, oldText string, newText string) (string, error) {
	path, err := util.SafePath(workingDirectory, filePath)

	if err != nil {
		log.Error("caminho inválido", "path", filePath, "err", err)
		return fmt.Sprintf("Error %s", err), fmt.Errorf("%w: %w", ErrOutsideRoot, err)
	}

	stats, err := os.Stat(path)
	if err != nil {
		return "Error: File not found.", ErrNotFound
	}

	if stats.Size() > int64(MAX_EDIT_FILE_SIZE) {
		return fmt.Sprintf("Error: File is too large. File has %s bu the limit is %s", util.FormatFileSize(stats.Size()), util.FormatFileSize(int64(MAX_EDIT_FILE_SIZE))), ErrTooLarge
	}

	content, err := os.ReadFile(path)
	if err != nil {
		log.Error("falha ao ler arquivo", "path", path, "err", err)
		return "Error: failed to read File content", fmt.Errorf("lendo o arquivo para editar: %w", err)
	}

	contentStr := string(content)
	switch occurrences := strings.Count(contentStr, oldText); {
	case occurrences == 0:
		return "Error: old_text not found in file", ErrTextNotFound
	case occurrences > 1:
		return fmt.Sprintf("%s%d", EDIT_MULTIPLE_MATCHES_NOTICE, occurrences), fmt.Errorf("%w: %d ocorrências", ErrTextNotUnique, occurrences)
	}

	statsAfterRead, err := os.Stat(path)
	if err != nil {
		log.Error("falha ao checar modificação do arquivo", "path", path, "err", err)
		return "Error: failed to check file staleness", fmt.Errorf("checando a modificação do arquivo: %w", err)
	}

	if !statsAfterRead.ModTime().Equal(stats.ModTime()) {
		return "Error: File was modified by another process since it was last accessed.", ErrStale
	}

	newContentStr := strings.Replace(contentStr, oldText, newText, 1)
	err = os.WriteFile(path, []byte(newContentStr), 0644)
	if err != nil {
		log.Error("falha ao escrever arquivo", "path", path, "err", err)
		return "Error: failed to write File content", fmt.Errorf("gravando o arquivo editado: %w", err)
	}

	return "Success: File edited", nil
}
