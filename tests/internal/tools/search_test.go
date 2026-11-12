package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	filetools "github.com/Gabriel-Araujo/network_agent/internal/tools/file"
)

func TestExpandBraces(t *testing.T) {
	tests := []struct {
		pattern  string
		expected []string
	}{
		{
			pattern:  "file.txt",
			expected: []string{"file.txt"},
		},
		{
			pattern:  "file.{txt,md}",
			expected: []string{"file.txt", "file.md"},
		},
		{
			pattern:  "*.{go,js,ts}",
			expected: []string{"*.go", "*.js", "*.ts"},
		},
		{
			pattern:  "{a,b}/{c,d}.txt",
			expected: []string{"a/c.txt", "a/d.txt", "b/c.txt", "b/d.txt"},
		},
		{
			pattern:  "no-end-{brace",
			expected: []string{"no-end-{brace"},
		},
	}

	for _, tt := range tests {
		result := filetools.ExpandBraces(tt.pattern)
		if len(result) != len(tt.expected) {
			t.Errorf("expandBraces(%q) returned %d items, want %d", tt.pattern, len(result), len(tt.expected))
			continue
		}
		for i := range result {
			if result[i] != tt.expected[i] {
				t.Errorf("expandBraces(%q) at index %d = %q, want %q", tt.pattern, i, result[i], tt.expected[i])
			}
		}
	}
}

func TestSearchFile(t *testing.T) {
	// As fixtures vão para um diretório temporário, nunca para a árvore do módulo:
	// um .go gerado aqui dentro quebra `go vet ./...` e `gofmt -l .` do repo inteiro.
	t.Chdir(t.TempDir())

	// Cria alguns arquivos e diretórios para testar
	files := []string{
		"a.txt",
		"b.md",
		"sub/c.go",
		"sub/d.txt",
		"another/e.go",
	}

	for _, f := range files {
		err := os.MkdirAll(filepath.Dir(f), 0755)
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(f, []byte("content"), 0644)
		if err != nil {
			t.Fatal(err)
		}
		// Espera um pouco para garantir tempos de modificação diferentes se necessário
		// Embora a ordenação padrão seja descendente, os arquivos criados rapidamente
		// podem ter o mesmo timestamp. Para testes de ordenação, precisaremos dar
		// tempos explícitos se quisermos testar a lógica de ordenação de forma confiável.
		time.Sleep(time.Millisecond * 10)
	}

	tests := []struct {
		name     string
		path     string
		pattern  string
		expected int
	}{
		{
			name:     "Search all txt files",
			path:     ".",
			pattern:  "*.txt",
			expected: 2, // a.txt, sub/d.txt (devido ao WalkDir recursivo e replace ** por *)
		},
		{
			name:     "Search all go files in sub",
			path:     "sub",
			pattern:  "*.go",
			expected: 1, // sub/c.go
		},
		{
			name:     "Search with braces",
			path:     ".",
			pattern:  "*.{txt,md}",
			expected: 3, // a.txt, b.md, sub/d.txt
		},
		{
			name:     "Search specific file",
			path:     ".",
			pattern:  "a.txt",
			expected: 1,
		},
		{
			name:     "Search non-existent",
			path:     ".",
			pattern:  "*.pdf",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _ := filetools.SearchPattern(".", tt.path, tt.pattern)
			if strings.HasPrefix(result, "Error") {
				t.Fatalf("SearchFile failed: %s", result)
			}

			var output filetools.GlobSearchOutput
			err := json.Unmarshal([]byte(result), &output)
			if err != nil {
				t.Fatalf("Failed to unmarshal output: %v", err)
			}

			if output.NumFiles != tt.expected {
				t.Errorf("Expected %d files, got %d. Files: %v", tt.expected, output.NumFiles, output.Filenames)
			}
		})
	}
}

func TestSearchFileSecurity(t *testing.T) {
	result, _ := filetools.SearchPattern(t.TempDir(), "../outside", "*.txt")
	if !strings.Contains(result, "Error") {
		t.Errorf("Expected security error, got: %s", result)
	}
}

func TestSearchFileOrdering(t *testing.T) {
	t.Chdir(t.TempDir())

	err := os.WriteFile("old.txt", []byte("old"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Garante que o arquivo mais novo tenha um tempo de modificação posterior
	time.Sleep(time.Second)

	err = os.WriteFile("new.txt", []byte("new"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	result, _ := filetools.SearchPattern(".", ".", "*.txt")
	var output filetools.GlobSearchOutput
	json.Unmarshal([]byte(result), &output)

	if len(output.Filenames) < 2 {
		t.Fatalf("Expected at least 2 files, got %d", len(output.Filenames))
	}

	// O mais novo (file2) deve vir primeiro
	if !strings.Contains(output.Filenames[0], "new.txt") {
		t.Errorf("Expected newest file first, but got %s", output.Filenames[0])
	}
}
