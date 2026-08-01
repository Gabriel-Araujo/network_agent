package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var nameRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

type Metadata struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	License     string         `yaml:"license,omitempty"`
	Extra       map[string]any `yaml:"metadata,omitempty"`
}

type Skill struct {
	Metadata
	Dir  string
	Body string
}

func validateName(name string) error {
	if len(name) == 0 || len(name) > 64 {
		return fmt.Errorf("nome deve ter entre 1 e 64 caracteres")
	}
	if !nameRe.MatchString(name) {
		return fmt.Errorf("nome deve conter só letras minúsculas, números e hífens simples")
	}
	return nil
}

func ParseSkillFile(path string) (*Skill, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("lendo %s: %w", path, err)
	}

	content := string(raw)
	if !strings.HasPrefix(strings.TrimSpace(content), "---") {
		return nil, fmt.Errorf("%s: frontmatter YAML ausente", path)
	}

	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("%s: frontmatter malformado (faltam os delimitadores ---)", path)
	}

	var meta Metadata
	if err := yaml.Unmarshal([]byte(parts[1]), &meta); err != nil {
		return nil, fmt.Errorf("%s: yaml inválido: %w", path, err)
	}
	if meta.Name == "" || meta.Description == "" {
		return nil, fmt.Errorf("%s: 'name' e 'description' são obrigatórios", path)
	}
	if err := validateName(meta.Name); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	return &Skill{
		Metadata: meta,
		Dir:      filepath.Dir(path),
		Body:     strings.TrimSpace(parts[2]),
	}, nil
}
