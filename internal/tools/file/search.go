package filetools

import (
	"encoding/json"
	"log"
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

func SearchPattern(workingDirectory, path, pattern string) string {
	start := time.Now()

	rel, err := util.SafePath(workingDirectory, path)
	if err != nil {
		log.Default().Print(err)
		return "Error: Path outside the permitted working directory"
	}

	patterns := ExpandBraces(pattern)
	var allMatches []string
	seen := make(map[string]bool)

	for _, pat := range patterns {
		// substitui ** por * para simplificar para filepath. Match,
		// já que o WalkDir já percorre recursivamente.
		matchPat := strings.ReplaceAll(pat, "**", "*")

		err := filepath.WalkDir(rel, func(path string, d os.DirEntry, err error) error {
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

				// Se ainda não deu match, tenta construir o caminho relativo ao diretório base 'rel'
				if !matched {
					if relPath, err := filepath.Rel(rel, path); err == nil {
						matched, _ = filepath.Match(matchPat, relPath)
					}
				}

				if matched {
					absMatch, _ := filepath.Abs(path)
					if !seen[absMatch] {
						allMatches = append(allMatches, path)
						seen[absMatch] = true
					}
				}
			}
			return nil
		})
		if err != nil {
			log.Default().Print(err)
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
		return "Error: failed to marshal search results"
	}

	return string(jsonData)
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
