package filetools

import (
	"fmt"
	"log"
	"os"

	"github.com/Gabriel-Araujo/network_agent/pkg/util"
)

func WriteFile(workingDirectory string, filePath string, content string) string {
	path, err := util.SafePath(workingDirectory, filePath)

	if err != nil {
		log.Default().Print(err)
		return fmt.Sprintf("Error %s", err)
	}

	_, err = os.Stat(path)
	if err == nil {
		return "Error: File found. Use file edit tool instead."
	}

	if len(content) > MAX_EDIT_FILE_SIZE {
		return fmt.Sprintf("Error: Content is too large. Content has %s bu the limit is %s", util.FormatFileSize(int64(len(content))), util.FormatFileSize(int64(MAX_EDIT_FILE_SIZE)))
	}

	err = os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		log.Default().Print(err)
		return "Error: failed to write File content"
	}

	return content
}
