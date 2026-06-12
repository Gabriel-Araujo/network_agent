package filetools

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

func ReadFile(workingDirectory string, filePath string) string {
	clean := filepath.Join(workingDirectory, filepath.Clean("/"+filePath))
	rel, err := filepath.Rel(workingDirectory, clean)

	if err != nil || strings.HasPrefix(rel, "..") {
		log.Default().Print(err)
		return "Error: Filepath outside the permitted working directory"
	}

	file, err := os.ReadFile(rel)
	if err != nil {
		log.Default().Print(err)
		return "Error: failed to read File content"
	}

	return string(file)
}
