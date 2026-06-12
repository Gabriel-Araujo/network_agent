package file_tools

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Gabriel-Araujo/network_agent/pkg/util"
)

func EditFile(working_directory string, file_path string, old_text string, new_text string) string {
	rel, err := util.SafePath(working_directory, file_path)

	if err != nil {
		log.Default().Print(err)
		return fmt.Sprintf("Error %s", err)
	}

	stats, err := os.Stat(rel)
	if err != nil {
		return "Error: File not found."
	}

	if stats.Size() > int64(MAX_EDIT_FILE_SIZE) {
		return fmt.Sprintf("Error: File is too large. File has %s bu the limit is %s", util.FormatFileSize(stats.Size()), util.FormatFileSize(int64(MAX_EDIT_FILE_SIZE)))
	}

	content, err := os.ReadFile(rel)
	if err != nil {
		log.Default().Print(err)
		return "Error: failed to read File content"
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, old_text) {
		return "Error: old_text not found in file"
	}

	statsAfterRead, err := os.Stat(rel)
	if err != nil {
		log.Default().Print(err)
		return "Error: failed to check file staleness"
	}

	if !statsAfterRead.ModTime().Equal(stats.ModTime()) {
		return "Error: File was modified by another process since it was last accessed."
	}

	newContentStr := strings.ReplaceAll(contentStr, old_text, new_text)
	err = os.WriteFile(rel, []byte(newContentStr), 0644)
	if err != nil {
		log.Default().Print(err)
		return "Error: failed to write File content"
	}

	return "Success: File edited"
}
