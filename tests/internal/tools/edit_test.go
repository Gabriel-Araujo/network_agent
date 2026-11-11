package tools

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
		got, _ := filetools.EditFile(".", fileName, "original", "changed")
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
		got, _ := filetools.EditFile(".", fileName, "not-there", "whatever")
		want := "Error: old_text not found in file"
		if got != want {
			t.Errorf("EditFile() = %q, want %q", got, want)
		}
	})

	t.Run("Staleness check", func(t *testing.T) {
		// Teste de concorrência omitido por simplicidade
	})

	t.Run("File not found", func(t *testing.T) {
		got, _ := filetools.EditFile(".", "missing.txt", "old", "new")
		if !strings.Contains(got, "File not found") {
			t.Errorf("EditFile() = %q, want error 'File not found'", got)
		}
	})

	t.Run("Security check", func(t *testing.T) {
		got, _ := filetools.EditFile(".", "../outside.txt", "old", "new")
		if !strings.Contains(got, "Error") {
			t.Errorf("EditFile() = %q, want error", got)
		}
	})

	t.Run("Multiple occurrences rejected", func(t *testing.T) {
		name := "two_neighbors.conf"
		initial := " neighbor 10.0.0.2 remote-as 65002\n neighbor 10.0.0.3 remote-as 65002\n"
		if err := os.WriteFile(name, []byte(initial), 0644); err != nil {
			t.Fatal(err)
		}

		got, _ := filetools.EditFile(".", name, "remote-as 65002", "remote-as 65003")
		want := filetools.EDIT_MULTIPLE_MATCHES_NOTICE + "2"
		if got != want {
			t.Errorf("EditFile() = %q, want %q", got, want)
		}

		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != initial {
			t.Errorf("file was modified: %q, want %q", string(data), initial)
		}

		// O mesmo pedido com texto em volta casa uma vez so e troca so esse vizinho.
		got, _ = filetools.EditFile(".", name, "10.0.0.2 remote-as 65002", "10.0.0.2 remote-as 65003")
		if got != "Success: File edited" {
			t.Errorf("EditFile() = %q, want %q", got, "Success: File edited")
		}

		data, err = os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		wantContent := " neighbor 10.0.0.2 remote-as 65003\n neighbor 10.0.0.3 remote-as 65002\n"
		if string(data) != wantContent {
			t.Errorf("Content = %q, want %q", string(data), wantContent)
		}
	})
}
