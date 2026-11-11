package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Gabriel-Araujo/network_agent/internal/config"
	"github.com/Gabriel-Araujo/network_agent/internal/tools"
	filetools "github.com/Gabriel-Araujo/network_agent/internal/tools/file"
	"github.com/openai/openai-go/v3/responses"
)

// TestForSubagentMountsTheConfiguration pins the minimal loader: the Subagente
// answers with its folder's AGENT.md mounted whole as the system prompt, the
// same connection and model as the generic agent, and the file tools plus
// read_skill_file — without the subagent tool (no nesting) and without the
// use_skill catalog.
func TestForSubagentMountsTheConfiguration(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "modifier")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("creating the subagent folder: %v", err)
	}
	const config = "# System Prompt — Modifier\n\nYou are the Modifier."
	if err := os.WriteFile(filepath.Join(dir, "AGENT.md"), []byte(config), 0o644); err != nil {
		t.Fatalf("writing AGENT.md: %v", err)
	}

	base := &Agent{ModelName: "model", systemPrompt: "generic prompt"}
	sub, err := base.forSubagentAt(root, "modifier")
	if err != nil {
		t.Fatalf("forSubagentAt returned an error: %v", err)
	}
	if sub == base {
		t.Fatal("returned the generic agent itself instead of a subagent copy")
	}
	if sub.systemPrompt != config {
		t.Errorf("system prompt %q, want %q", sub.systemPrompt, config)
	}
	if sub.ModelName != "model" {
		t.Errorf("model %q, want %q — the Subagente inherits the connection", sub.ModelName, "model")
	}

	tools, err := json.Marshal(sub.availableTools)
	if err != nil {
		t.Fatalf("marshaling the tools: %v", err)
	}
	if !strings.Contains(string(tools), `"read_skill_file"`) || !strings.Contains(string(tools), `"ReadFile"`) {
		t.Errorf("subagent tools are missing file/skills tools: %s", tools)
	}
	if strings.Contains(string(tools), `"use_skill"`) {
		t.Errorf("subagent inherited the use_skill catalog: %s", tools)
	}
	if strings.Contains(string(tools), `"sub_agent"`) {
		t.Errorf("subagent inherited the subagent tool (nested routing): %s", tools)
	}
}

// TestForSubagentExplicitErrors: a missing folder or AGENT.md — or an empty
// one — is an explicit error, never a silently empty system prompt.
func TestForSubagentExplicitErrors(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "only-folder"), 0o755); err != nil {
		t.Fatalf("creating a folder without AGENT.md: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "empty-config"), 0o755); err != nil {
		t.Fatalf("creating a folder with an empty AGENT.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "empty-config", "AGENT.md"), []byte("  \n"), 0o644); err != nil {
		t.Fatalf("writing the empty AGENT.md: %v", err)
	}

	base := &Agent{}
	cases := []struct{ name, want string }{
		{"", "no name"},               // empty name
		{"ghost", "not found"},        // missing folder
		{"only-folder", "not loaded"}, // missing AGENT.md
		{"empty-config", "empty"},     // empty AGENT.md
	}
	for _, c := range cases {
		sub, err := base.forSubagentAt(root, c.name)
		if err == nil {
			t.Errorf("forSubagentAt(%q) returned agent %v, want an error", c.name, sub)
			continue
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Errorf("forSubagentAt(%q) error %q, want it to contain %q", c.name, err, c.want)
		}
	}
}

// TestSubagentToolsetDeniesWriteFileToTheModifier pins the per-name toolset:
// the Modifier is dispatched without WriteFile, and the other three Subagentes
// keep the whole dispatched toolset. None of them gets the subagent tool.
func TestSubagentToolsetDeniesWriteFileToTheModifier(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"builder", "modifier", "teacher", "illustrator"} {
		dir := filepath.Join(root, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("creating the %s folder: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "AGENT.md"), []byte("You are the "+name+"."), 0o644); err != nil {
			t.Fatalf("writing %s/AGENT.md: %v", name, err)
		}
	}

	whole := toolNames(tools.GetFileAndSkillTools())
	if !slices.Contains(whole, filetools.WRITE_TOOL_NAME) {
		t.Fatalf("the dispatched toolset no longer has %s at all: %v", filetools.WRITE_TOOL_NAME, whole)
	}
	withoutWrite := slices.DeleteFunc(slices.Clone(whole), func(n string) bool { return n == filetools.WRITE_TOOL_NAME })

	base := &Agent{}
	for _, name := range []string{"builder", "modifier", "teacher", "illustrator"} {
		sub, err := base.forSubagentAt(root, name)
		if err != nil {
			t.Fatalf("forSubagentAt(%q): %v", name, err)
		}
		got := toolNames(sub.availableTools)

		want := whole
		if name == "modifier" {
			want = withoutWrite
		}
		if !slices.Equal(got, want) {
			t.Errorf("toolset of %s is %v, want %v", name, got, want)
		}
		if slices.Contains(got, tools.SUB_AGENT_TOOL_NAME) {
			t.Errorf("%s inherited %s (nested routing): %v", name, tools.SUB_AGENT_TOOL_NAME, got)
		}
	}
}

// toolNames lists the function names a toolset offers the model.
func toolNames(toolset []responses.ToolUnionParam) []string {
	names := make([]string, 0, len(toolset))
	for _, t := range toolset {
		if t.OfFunction != nil {
			names = append(names, t.OfFunction.Name)
		}
	}
	return names
}

// TestForSubagentResolvesInTheMirror pins the reader half of the mirror: the
// real ForSubagent — not the forSubagentAt seam the tests above drive — mounts
// the AGENT.md that lives under the Agent root in the user's home, projected
// there from the binary, and nothing in the path depends on the source tree
// existing. Driving "teacher" covers both Go readers at once: mountPromptSkills
// calls LoadSkills for the mermaid skill, which resolves in the same mirror.
//
// This is the call the live drive of ticket 07 found reaching into the source
// tree through runtime.Caller.
func TestForSubagentResolvesInTheMirror(t *testing.T) {
	sub, err := (&Agent{ModelName: "model"}).ForSubagent("teacher")
	if err != nil {
		t.Fatalf("ForSubagent(teacher): %v", err)
	}

	mirrored, err := config.MaterialFile("resources/agents/teacher/AGENT.md")
	if err != nil {
		t.Fatalf("locating the mirrored AGENT.md: %v", err)
	}
	raw, err := os.ReadFile(mirrored)
	if err != nil {
		t.Fatalf("reading the mirrored AGENT.md: %v", err)
	}
	if !strings.HasPrefix(sub.systemPrompt, string(raw)) {
		t.Errorf("the mounted prompt does not start with %s", mirrored)
	}
	// The mermaid skill is appended after it, so the prompt is strictly longer.
	if len(sub.systemPrompt) <= len(raw) {
		t.Errorf("prompt of %d bytes, want the AGENT.md (%d) plus the mounted skill", len(sub.systemPrompt), len(raw))
	}
}
