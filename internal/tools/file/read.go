package file_tools

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

func ReadFile(working_directory string, path string) string {
	clean := filepath.Join(working_directory, filepath.Clean("/"+path))
	rel, err := filepath.Rel(working_directory, clean)

	if err != nil || strings.HasPrefix(rel, "..") {
		log.Default().Print(err)
		return "Error: Filepath outside the permiteed working directory"
	}

	file, err := os.ReadFile(rel)
	if err != nil {
		log.Default().Print(err)
		return "Error: failed to read File content"
	}

	return string(file)
}
