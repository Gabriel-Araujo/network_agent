package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestActivationDeliversTheTurnPack pins the result of use_skill against the
// real skills (.agent/frr-file-builder/issues/06-decidir-como-o-pack-chega-ao-modelo.md):
// frr-file-builder is its body plus the pack of the turn's protocol, or the
// line that says why there is none, plus the RAG reach line when the turn
// retrieved for a protocol; mermaid is its body and nothing else.
func TestActivationDeliversTheTurnPack(t *testing.T) {
	reg := LoadSkills()
	pack := []string{"references/bgp/TECH.md", "references/bgp/syntax.md", "references/bgp/doctrine.md"}
	const (
		reach  = "retrieved for "
		absent = "! WARNING: no pack covers ospf; built from the retrieved FRR documentation only"
		empty  = "This turn has no classified technology"
	)

	for _, tc := range []struct {
		name, skill string
		turn        Turn
		wantFiles   bool
		wantLines   []string
		denyLines   []string
	}{
		{name: "bgp with context", skill: packSkill, turn: Turn{"bgp", true}, wantFiles: true,
			wantLines: []string{reach + "bgp only"}, denyLines: []string{empty, "no pack covers"}},
		{name: "bgp without context", skill: packSkill, turn: Turn{"bgp", false}, wantFiles: true,
			denyLines: []string{reach, empty, "no pack covers"}},
		{name: "protocol without pack", skill: packSkill, turn: Turn{"ospf", false},
			wantLines: []string{absent}, denyLines: []string{reach, empty}},
		{name: "empty protocol", skill: packSkill, turn: Turn{"", true},
			wantLines: []string{empty}, denyLines: []string{reach, "no pack covers"}},
		{name: "mermaid", skill: "mermaid", turn: Turn{"bgp", true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sk, ok := reg.Get(tc.skill)
			if !ok {
				t.Fatalf("skill %s not loaded", tc.skill)
			}
			got, err := reg.activation(sk, tc.turn)
			if err != nil {
				t.Fatalf("activation: %v", err)
			}
			if !strings.HasPrefix(got, sk.Body) {
				t.Fatalf("result does not open with the body: %.120q", got)
			}
			if tc.skill == "mermaid" && got != sk.Body {
				t.Errorf("mermaid result = body + %q, want the body alone", got[len(sk.Body):])
			}

			at := len(sk.Body)
			for _, rel := range pack {
				header := `<file path="` + rel + `">`
				i := strings.Index(got, header)
				if !tc.wantFiles {
					if i >= 0 {
						t.Errorf("result carries %s, want no pack file", rel)
					}
					continue
				}
				if i < at {
					t.Errorf("%s missing or out of order (TECH.md, syntax.md, doctrine.md)", rel)
					continue
				}
				content, err := reg.ReadFile(packSkill, rel)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.HasPrefix(got[i+len(header)+1:], strings.TrimRight(content, "\n")+"\n</file>") {
					t.Errorf("%s is not delivered whole right after its path", rel)
				}
				at = i
			}
			for _, line := range tc.wantLines {
				if !strings.Contains(got[len(sk.Body):], line) {
					t.Errorf("result lacks %q", line)
				}
			}
			for _, line := range tc.denyLines {
				if strings.Contains(got[len(sk.Body):], line) {
					t.Errorf("result carries %q, want it absent", line)
				}
			}
		})
	}
}

// TestActivationIncompletePackIsAnError: a pack folder missing one of the
// three files fails the activation and names the file — never a partial pack
// the model would build from believing it has the rules.
func TestActivationIncompletePackIsAnError(t *testing.T) {
	root := t.TempDir()
	for path, content := range map[string]string{
		packSkill + "/SKILL.md":                      "---\nname: " + packSkill + "\ndescription: test skill\n---\nbody",
		packSkill + "/references/bgp/TECH.md":        "tech",
		packSkill + "/references/bgp/doctrine.md":    "doctrine",
		packSkill + "/references/ospf/examples/x.md": "an example is not a pack",
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
	sk, _ := reg.Get(packSkill)

	for protocol, missing := range map[string]string{"bgp": "syntax.md", "ospf": "TECH.md"} {
		got, err := reg.activation(sk, Turn{Protocol: protocol})
		if err == nil {
			t.Errorf("%s: activation = %q, want an error", protocol, got)
			continue
		}
		if !strings.Contains(err.Error(), missing) {
			t.Errorf("%s: err = %v, want %s named", protocol, err, missing)
		}
	}
}
