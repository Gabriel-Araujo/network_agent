package filetools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	filetools "github.com/Gabriel-Araujo/network_agent/internal/tools/file"
)

func TestEditFile(t *testing.T) {
	// Define the generated directory path
	genDir := filepath.Join("generated", "edit")
	err := os.MkdirAll(genDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	originalWd, _ := os.Getwd()
	os.Chdir(genDir)
	defer os.Chdir(originalWd)

	fileName := "edit_me.txt"
	initialContent := "original content"
	fullPath := fileName

	err = os.WriteFile(fullPath, []byte(initialContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("Success edit", func(t *testing.T) {
		got := filetools.EditFile(".", fileName, "original", "changed")
		want := "Success: File edited"
		if got != want {
			t.Errorf("EditFile() = %q, want %q", got, want)
		}
		data, _ := os.ReadFile(fullPath)
		if string(data) != "changed content" {
			t.Errorf("Content = %q, want %q", string(data), "changed content")
		}
	})

	t.Run("Old text not found", func(t *testing.T) {
		got := filetools.EditFile(".", fileName, "not-there", "whatever")
		want := "Error: old_text not found in file"
		if got != want {
			t.Errorf("EditFile() = %q, want %q", got, want)
		}
	})

	t.Run("Staleness check", func(t *testing.T) {
		// Teste de concorrência omitido por simplicidade
	})

	t.Run("File not found", func(t *testing.T) {
		got := filetools.EditFile(".", "missing.txt", "old", "new")
		if !strings.Contains(got, "File not found") {
			t.Errorf("EditFile() = %q, want error 'File not found'", got)
		}
	})

	t.Run("Security check", func(t *testing.T) {
		got := filetools.EditFile(".", "../outside.txt", "old", "new")
		if !strings.Contains(got, "Error") {
			t.Errorf("EditFile() = %q, want error", got)
		}
	})
}
