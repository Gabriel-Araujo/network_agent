package llm

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	llm2 "github.com/Gabriel-Araujo/network_agent/internal/agent"
	"github.com/Gabriel-Araujo/network_agent/internal/agent/rag"
	"github.com/Gabriel-Araujo/network_agent/internal/paths"
	filetools "github.com/Gabriel-Araujo/network_agent/internal/tools/file"
)

// TestAgentResourcesFitTheReadCap guards the subagents' support material
// (tech packs, stage files): ReadFile returns only the first
// READ_MAX_CHARS_TO_READ bytes, so a file past the cap would reach the model
// cut while every other gate stays green.
func TestAgentResourcesFitTheReadCap(t *testing.T) {
	root, err := paths.RepoFile("resources/agents")
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".md") {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		checked++
		if info.Size() > filetools.READ_MAX_CHARS_TO_READ {
			t.Errorf("%s has %d bytes, past the ReadFile cap of %d", path, info.Size(), filetools.READ_MAX_CHARS_TO_READ)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if checked == 0 {
		t.Fatalf("no .md file found under %s", root)
	}
}

// TestSubagentConfigsOnDisk proves the name -> folder table of the routing
// List: every value of rag.SubagentListName loads its AGENT.md from
// resources/agents/<name>/. The List rows are the single source of that table,
// so a value with no configuration fails here and nowhere else.
//
// It also asserts that rag.DefaultSubagent is one of those values, which is
// what keeps the turn's last resort out of reach: a default that is not in the
// List, or whose configuration is missing, is the one case that kills a turn
// (decision 4 of .agent/generico/issues/01-quem-responde-o-nenhum.md), and it
// is barred here instead of live.
func TestSubagentConfigsOnDisk(t *testing.T) {
	base := &llm2.Agent{}
	found := false
	defaultInList := false
	for _, l := range rag.Lists() {
		if l.Name != rag.SubagentListName {
			continue
		}
		found = true
		if len(l.Values) == 0 {
			t.Fatal("the routing List has no values")
		}
		for _, v := range l.Values {
			if v.Name == rag.DefaultSubagent {
				defaultInList = true
			}
			if _, err := base.ForSubagent(v.Name); err != nil {
				t.Errorf("subagent %q has no usable configuration: %v", v.Name, err)
			}
		}
	}
	if !found {
		t.Fatalf("List %q not found in rag.Lists()", rag.SubagentListName)
	}
	if !defaultInList {
		t.Errorf("the default Subagente %q is not a value of List %q", rag.DefaultSubagent, rag.SubagentListName)
	}
}
