package filetools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	filetools "github.com/Gabriel-Araujo/network_agent/internal/tools/file"
)

func TestReadFile(t *testing.T) {
	// Define the generated directory path
	genDir := filepath.Join("generated", "read")
	err := os.MkdirAll(genDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	originalWd, _ := os.Getwd()
	os.Chdir(genDir)
	defer os.Chdir(originalWd)

	fileName := "test.txt"
	content := "hello world"

	err = os.WriteFile(fileName, []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		workingDir string
		filePath   string
		want       string
	}{
		{
			name:       "Success read",
			workingDir: ".",
			filePath:   fileName,
			want:       "hello world",
		},
		{
			name:       "Path outside",
			workingDir: ".",
			filePath:   "../outside.txt",
			want:       "Error: Filepath outside the permitted working directory",
		},
		{
			name:       "File not found",
			workingDir: ".",
			filePath:   "nonexistent.txt",
			want:       "Error: failed to read File content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filetools.ReadFile(tt.workingDir, tt.filePath)
			if tt.name == "Success read" {
				if got != tt.want {
					t.Errorf("ReadFile() = %q, want %q", got, tt.want)
				}
			} else {
				if !strings.HasPrefix(got, "Error") {
					t.Errorf("ReadFile() = %q, want error message starting with 'Error'", got)
				}
				if got != tt.want {
					t.Errorf("ReadFile() = %q, want %q", got, tt.want)
				}
			}
		})
	}
}
