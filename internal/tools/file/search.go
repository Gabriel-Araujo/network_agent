package filetools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Gabriel-Araujo/network_agent/pkg/util"
)

type GlobSearchOutput struct {
	DurationMs int64    `json:"duration_ms"`
	NumFiles   int      `json:"num_files"`
	Filenames  []string `json:"filenames"`
	Truncated  bool     `json:"truncated"`
}

// SearchPattern lista os arquivos sob path que casam com pattern, resolvido
// contra workingDirectory. Devolve o que o modelo lê e a falha tipada (ver
// ReadFile).
func SearchPattern(workingDirectory, path, pattern string) (string, error) {
	start := time.Now()

	root, err := util.SafePath(workingDirectory, path)
	if err != nil {
		log.Error("caminho fora do diretório de trabalho", "path", path, "err", err)
		return "Error: Path outside the permitted working directory", fmt.Errorf("%w: %w", ErrOutsideRoot, err)
	}

	patterns := ExpandBraces(pattern)
	var allMatches []string
	seen := make(map[string]bool)

	for _, pat := range patterns {
		// substitui ** por * para simplificar para filepath. Match,
		// já que o WalkDir já percorre recursivamente.
		matchPat := strings.ReplaceAll(pat, "**", "*")

		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if !d.IsDir() {
				// Tenta match com o caminho relativo completo
				matched, _ := filepath.Match(matchPat, path)

				// Se não deu match e o padrão não tem separadores, tenta match apenas no nome do arquivo
				if !matched && !strings.Contains(matchPat, string(filepath.Separator)) {
					matched, _ = filepath.Match(matchPat, filepath.Base(path))
				}

				// Se ainda não deu match, tenta construir o caminho relativo ao diretório base 'root'
				if !matched {
					if relPath, err := filepath.Rel(root, path); err == nil {
						matched, _ = filepath.Match(matchPat, relPath)
					}
				}

				if matched {
					// root é absoluto, logo o path do WalkDir também:
					// serve de chave de deduplicação entre os padrões
					// expandidos sem passar por filepath.Abs.
					if !seen[path] {
						allMatches = append(allMatches, path)
						seen[path] = true
					}
				}
			}
			return nil
		})
		if err != nil {
			log.Error("falha ao percorrer diretório", "root", root, "pattern", pat, "err", err)
		}
	}

	// Ordenar por data de modificação (mais recente primeiro)
	slices.SortFunc(allMatches, func(a, b string) int {
		infoA, _ := os.Stat(a)
		infoB, _ := os.Stat(b)
		var timeA, timeB time.Time
		if infoA != nil {
			timeA = infoA.ModTime()
		}
		if infoB != nil {
			timeB = infoB.ModTime()
		}
		if timeA.After(timeB) {
			return -1
		}
		if timeA.Before(timeB) {
			return 1
		}
		return 0
	})

	truncated := len(allMatches) > 100
	if truncated {
		allMatches = allMatches[:100]
	}

	output := GlobSearchOutput{
		DurationMs: time.Since(start).Milliseconds(),
		NumFiles:   len(allMatches),
		Filenames:  allMatches,
		Truncated:  truncated,
	}

	jsonData, err := json.Marshal(output)
	if err != nil {
		return "Error: failed to marshal search results", fmt.Errorf("serializando o resultado da busca: %w", err)
	}

	return string(jsonData), nil
}

func ExpandBraces(pattern string) []string {
	// Implementação simples de expansão de chaves {a, b}
	// Se não houver chaves, retorna o padrão original
	start := strings.Index(pattern, "{")
	end := strings.Index(pattern, "}")

	if start == -1 || end == -1 || start > end {
		return []string{pattern}
	}

	prefix := pattern[:start]
	suffix := pattern[end+1:]
	options := strings.Split(pattern[start+1:end], ",")

	var expanded []string
	for _, opt := range options {
		expanded = append(expanded, ExpandBraces(prefix+opt+suffix)...)
	}

	return expanded
}
