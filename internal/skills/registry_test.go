package skills

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// TestListDir pins read_skill_directory's listing: recursive, relative to the
// skill root with "/" on any OS — the path read_skill_file takes as is —,
// folders ending in "/", and the refusals a path outside a folder of the
// skill gets: ".." that would leave it, a missing folder, a file.
func TestListDir(t *testing.T) {
	root := t.TempDir()
	for path, content := range map[string]string{
		"demo/SKILL.md":                        "---\nname: demo\ndescription: test skill\n---\nbody",
		"demo/references/CONTEXT.md":           "context",
		"demo/references/bgp/TECH.md":          "tech",
		"demo/references/bgp/examples/ebgp.md": "example",
		"demo/references/bgp/examples/ibgp.md": "example",
	} {
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	reg, err := LoadDir(root)
	if err != nil {
		t.Fatal(err)
	}

	references := []string{
		"references/CONTEXT.md",
		"references/bgp/",
		"references/bgp/TECH.md",
		"references/bgp/examples/",
		"references/bgp/examples/ebgp.md",
		"references/bgp/examples/ibgp.md",
	}
	for _, tc := range []struct {
		path string
		want []string
	}{
		{"references", references},
		{"references/", references},
		{"", append([]string{"SKILL.md", "references/"}, references...)},
		{".", append([]string{"SKILL.md", "references/"}, references...)},
		{"references/bgp/examples", references[4:]},
	} {
		got, err := reg.ListDir("demo", tc.path)
		if err != nil {
			t.Errorf("ListDir(%q): %v", tc.path, err)
			continue
		}
		if !slices.Equal(got, tc.want) {
			t.Errorf("ListDir(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}

	for _, path := range []string{"..", "../demo", "references/../..", "references/missing", "SKILL.md"} {
		if got, err := reg.ListDir("demo", path); err == nil {
			t.Errorf("ListDir(%q) = %v, want an error", path, got)
		}
	}
	if _, err := reg.ListDir("no-such-skill", ""); err == nil {
		t.Error("ListDir accepted a skill the registry does not have")
	}
}
