package filetools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	filetools "github.com/Gabriel-Araujo/network_agent/internal/tools/file"
)

func TestWriteFile(t *testing.T) {
	// Define the generated directory path
	genDir := filepath.Join("generated", "write")
	err := os.MkdirAll(genDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Clean up generated files from previous runs to avoid "File found" errors
	files, _ := filepath.Glob(filepath.Join(genDir, "*"))
	for _, f := range files {
		os.RemoveAll(f)
	}

	originalWd, _ := os.Getwd()
	os.Chdir(genDir)
	defer os.Chdir(originalWd)

	tests := []struct {
		name       string
		workingDir string
		filePath   string
		content    string
		want       string
	}{
		{
			name:       "Success write new file",
			workingDir: ".",
			filePath:   "new.txt",
			content:    "new content",
			want:       "new content",
		},
		{
			name:       "Success write in new subdirectory",
			workingDir: ".",
			filePath:   "sub/dir/new.txt",
			content:    "sub content",
			want:       "sub content",
		},
		{
			name:       "Error file already exists",
			workingDir: ".",
			filePath:   "existing.txt",
			content:    "will fail",
			want:       "Error: File found. Use file edit tool instead.",
		},
		{
			name:       "Error path outside",
			workingDir: ".",
			filePath:   "../outside.txt",
			content:    "fail",
			want:       "Error Filepath outside the permiteed working directory",
		},
	}

	// Setup existing file for the error case
	os.WriteFile("existing.txt", []byte("exists"), 0644)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filetools.WriteFile(tt.workingDir, tt.filePath, tt.content)
			if strings.HasPrefix(tt.want, "Error") {
				if got != tt.want {
					t.Errorf("WriteFile() = %q, want %q", got, tt.want)
				}
			} else {
				if got != tt.want {
					t.Errorf("WriteFile() = %q, want %q", got, tt.want)
				}
				// Verify file actually written
				data, err := os.ReadFile(filepath.Join(tt.workingDir, tt.filePath))
				if err != nil {
					t.Errorf("Failed to read back written file: %v", err)
				}
				if string(data) != tt.content {
					t.Errorf("File content = %q, want %q", string(data), tt.content)
				}
			}
		})
	}
}
