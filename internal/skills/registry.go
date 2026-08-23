package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Registry struct {
	skills map[string]*Skill
}

func (r *Registry) Get(name string) (*Skill, bool) {
	sk, ok := r.skills[name]
	return sk, ok
}

func (r *Registry) All() []*Skill {
	out := make([]*Skill, 0, len(r.skills))
	for _, s := range r.skills {
		out = append(out, s)
	}
	return out
}

// ReadFile lê um arquivo bundled da skill (scripts/, references/, assets/...),
// bloqueando qualquer tentativa de sair da pasta da skill via "..".
func (r *Registry) ReadFile(skillName, relPath string) (string, error) {
	sk, ok := r.skills[skillName]
	if !ok {
		return "", fmt.Errorf("skill %q não encontrada", skillName)
	}

	cleaned := filepath.Clean(string(filepath.Separator) + relPath)
	full := filepath.Join(sk.Dir, cleaned)

	if !strings.HasPrefix(full, filepath.Clean(sk.Dir)+string(filepath.Separator)) {
		return "", fmt.Errorf("caminho inválido: %q tenta sair da pasta da skill", relPath)
	}

	data, err := os.ReadFile(full)
	if err != nil {
		return "", fmt.Errorf("lendo %s: %w", relPath, err)
	}
	return string(data), nil
}

func LoadDir(root string) (*Registry, error) {
	reg := &Registry{skills: make(map[string]*Skill)}

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("lendo diretório de skills %s: %w", root, err)
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillPath := filepath.Join(root, e.Name(), "SKILL.md")
		if _, err := os.Stat(skillPath); err != nil {
			continue // pasta sem SKILL.md: ignora
		}

		sk, err := ParseSkillFile(skillPath)
		if err != nil {
			log.Warn("ignorando skill inválida", "path", skillPath, "err", err)
			continue
		}
		if sk.Name != e.Name() {
			log.Warn("'name' difere do nome da pasta",
				"path", skillPath, "name", sk.Name, "dir", e.Name())
		}
		reg.skills[sk.Name] = sk
	}

	return reg, nil
}
