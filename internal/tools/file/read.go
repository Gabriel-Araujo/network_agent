package filetools

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/Gabriel-Araujo/network_agent/pkg/util"
)

// ReadFile lê o arquivo em filePath, resolvido contra workingDirectory.
//
// Devolve o que o modelo lê **e** a falha tipada: a string é a mesma de antes
// (ela é contrato com o modelo), e o erro é o canal do Aviso. Os dois sempre
// juntos — uma string de erro sem erro é a falha calada que o ticket 14 mediu.
func ReadFile(workingDirectory string, filePath string) (string, error) {
	path, err := util.SafePath(workingDirectory, filePath)
	if err != nil {
		log.Error("caminho fora do diretório de trabalho", "path", filePath, "err", err)
		return "Error: Filepath outside the permitted working directory", fmt.Errorf("%w: %w", ErrOutsideRoot, err)
	}

	log.Trace("lendo arquivo", "path", path)
	file, err := os.ReadFile(path)
	if err != nil {
		log.Error("falha ao ler arquivo", "path", path, "err", err)
		if errors.Is(err, fs.ErrNotExist) {
			return READ_NOT_FOUND_NOTICE + path, ErrNotFound
		}
		return "Error: failed to read File content", fmt.Errorf("lendo o arquivo: %w", err)
	}

	content := string(file)
	if len(content) > READ_MAX_CHARS_TO_READ {
		content = content[:READ_MAX_CHARS_TO_READ] + "\n... (conteúdo truncado)"
	}

	return content, nil
}
