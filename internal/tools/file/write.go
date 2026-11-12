package filetools

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Gabriel-Araujo/network_agent/pkg/util"
)

// WriteFile grava content em filePath, resolvido contra workingDirectory,
// criando o diretório que falte. Devolve o que o modelo lê e a falha tipada
// (ver ReadFile).
func WriteFile(workingDirectory string, filePath string, content string) (string, error) {
	path, err := util.SafePath(workingDirectory, filePath)

	if err != nil {
		log.Error("caminho inválido", "path", filePath, "err", err)
		return fmt.Sprintf("Error %s", err), fmt.Errorf("%w: %w", ErrOutsideRoot, err)
	}

	_, err = os.Stat(path)
	replaced := err == nil

	if len(content) > MAX_EDIT_FILE_SIZE {
		return fmt.Sprintf("Error: Content is too large. Content has %s bu the limit is %s", util.FormatFileSize(int64(len(content))), util.FormatFileSize(int64(MAX_EDIT_FILE_SIZE))), ErrTooLarge
	}

	dir := filepath.Dir(path)
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		log.Error("falha ao criar diretório", "dir", dir, "err", err)
		return "Error: failed to create directory", fmt.Errorf("criando o diretório %s: %w", dir, err)
	}

	err = os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		log.Error("falha ao escrever arquivo", "path", path, "err", err)
		return "Error: failed to write File content", fmt.Errorf("gravando o arquivo: %w", err)
	}

	if replaced {
		return WRITE_REPLACED_NOTICE + filepath.ToSlash(path) + "\n" + content, nil
	}
	return WRITE_CREATED_NOTICE + filepath.ToSlash(path) + "\n" + content, nil
}
