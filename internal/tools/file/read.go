package filetools

import (
	"errors"
	"io/fs"
	"os"

	"github.com/Gabriel-Araujo/network_agent/pkg/util"
)

func ReadFile(workingDirectory string, filePath string) string {
	path, err := util.SafePath(workingDirectory, filePath)
	if err != nil {
		log.Error("caminho fora do diretório de trabalho", "path", filePath, "err", err)
		return "Error: Filepath outside the permitted working directory"
	}

	log.Trace("lendo arquivo", "path", path)
	file, err := os.ReadFile(path)
	if err != nil {
		log.Error("falha ao ler arquivo", "path", path, "err", err)
		if errors.Is(err, fs.ErrNotExist) {
			return READ_NOT_FOUND_NOTICE + path
		}
		return "Error: failed to read File content"
	}

	content := string(file)
	if len(content) > READ_MAX_CHARS_TO_READ {
		content = content[:READ_MAX_CHARS_TO_READ] + "\n... (conteúdo truncado)"
	}

	return content
}
