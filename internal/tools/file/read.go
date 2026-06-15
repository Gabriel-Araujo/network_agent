package filetools

import (
	"log"
	"os"
	"strings"

	"github.com/Gabriel-Araujo/network_agent/pkg/util"
)

func ReadFile(workingDirectory string, filePath string) string {
	rel, err := util.SafePath(workingDirectory, filePath)
	if err != nil || strings.HasPrefix(rel, "..") {
		log.Default().Print(err)
		return "Error: Filepath outside the permitted working directory"
	}

	log.Default().Print("Reading file: ", rel)
	file, err := os.ReadFile(rel)
	if err != nil {
		log.Default().Print(err)
		return "Error: failed to read File content"
	}

	content := string(file)
	if len(content) > READ_MAX_CHARS_TO_READ {
		content = content[:READ_MAX_CHARS_TO_READ] + "\n... (conteúdo truncado)"
	}

	return content
}
