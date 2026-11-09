package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	llm2 "github.com/Gabriel-Araujo/network_agent/internal/agent"
	"github.com/Gabriel-Araujo/network_agent/internal/skills"
	"github.com/Gabriel-Araujo/network_agent/internal/tools"
	"github.com/openai/openai-go/v3/responses"
)

// TestMermaidSkillInCatalog pins the skill the Illustrator and Teacher configs
// name: "mermaid" loads from resources/agents/skills, use_skill enumerates it,
// and read_skill_file serves one reference per covered diagram type. A broken
// frontmatter is only a logged warning in LoadDir, so a missing entry here is
// how it shows.
func TestMermaidSkillInCatalog(t *testing.T) {
	t.Chdir(repoRoot(t))

	reg := skills.LoadSkills()
	if _, ok := reg.Get("mermaid"); !ok {
		t.Fatal(`skill "mermaid" not loaded from resources/agents/skills`)
	}

	if enum := useSkillEnum(t, reg.BuildTools()); !slices.Contains(enum, "mermaid") {
		t.Errorf("use_skill does not enumerate mermaid: %v", enum)
	}

	a := &llm2.Agent{}
	refs := map[string]string{
		"references/flowchart.md": "flowchart",
		"references/sequence.md":  "sequenceDiagram",
		"references/state.md":     "stateDiagram-v2",
	}
	for path, header := range refs {
		call := responses.ResponseFunctionToolCall{
			CallID:    "call_read",
			Name:      skills.ToolReadSkillFile,
			Arguments: `{"skill_name": "mermaid", "relative_path": "` + path + `"}`,
		}
		got := callOutput(t, a.SkillCall(context.Background(), call))
		if strings.HasPrefix(got, "erro:") {
			t.Errorf("read_skill_file %s: %s", path, got)
			continue
		}
		// Normalize line endings: a Windows checkout may hand the file back in CRLF.
		if !strings.Contains(strings.ReplaceAll(got, "\r\n", "\n"), "```mermaid\n"+header) {
			t.Errorf("read_skill_file %s: no mermaid example starting with %q", path, header)
		}
	}
}

// pipelineSkills are the skills a dispatched Subagente has no use for: the
// frr-file-builder stub, the generic's intent analyzer, and the RAG retriever
// whose briefing input no live prompt produces any more (the turn's RAG is Go,
// internal/turn). Their descriptions order activation on any FRR question, so
// a dispatched Subagente that can read the catalog spends calls on them: 12 of
// 30 in the ticket 28 batch, and t11 never reached mermaid.
var pipelineSkills = []string{"frr-file-builder", "frr-intent-analyzer", "frr-rag-retriever"}

// TestNoSkillCatalogForTheSubagente pins both halves of the cut that ticket 18
// proved for the stub alone, extended to every skill: the toolset of a
// dispatched Subagente carries no use_skill at all — so no catalog, no enum,
// no description naming a skill — while the generic agent's still carries the
// whole catalog, which is out of this effort's scope and must not regress.
//
// The cut is the tool, not a filter on the registry, because after removing
// the three pipeline skills the only entry left was "mermaid", already mounted
// in the prompt of the two Subagentes who draw (promptSkills) — so use_skill
// returned text the prompt already had, a call the e4b spends 9 of 13 times
// for nothing (ticket 28 measured no discrimination: 3 OK/6 FAIL activated vs
// 1 OK/3 FAIL not).
func TestNoSkillCatalogForTheSubagente(t *testing.T) {
	t.Chdir(repoRoot(t))

	dispatched := tools.GetFileAndSkillTools()

	// Half one: the dispatched Subagente has no use_skill. A tool absent from
	// the request has no description for the model to read and no enum for it
	// to name, and callTool refuses the name anyway (ticket 20).
	if fn := findTool(dispatched, skills.ToolUseSkill); fn != nil {
		t.Errorf("the dispatched Subagente still offers %s: %v", skills.ToolUseSkill, skillEnum(t, fn))
	}

	// What must survive the cut: read_skill_file is how the Illustrator and
	// Teacher configs reach references/flowchart.md and its two siblings, and
	// both name the tool and the "mermaid" skill explicitly, so the reach does
	// not depend on a catalog.
	if findTool(dispatched, skills.ToolReadSkillFile) == nil {
		t.Errorf("the cut took %s with it: %v", skills.ToolReadSkillFile, toolNames(dispatched))
	}

	// No skill name reaches the dispatched model through any other tool
	// either: the catalog text was the carrier, and this is the half ticket 18
	// checked on the description.
	blob, err := json.Marshal(dispatched)
	if err != nil {
		t.Fatalf("marshaling the dispatched toolset: %v", err)
	}
	for _, name := range append(slices.Clone(pipelineSkills), "mermaid") {
		if bytes.Contains(blob, []byte(name)) {
			t.Errorf("the dispatched toolset still names the skill %q: %s", name, blob)
		}
	}

	// Half two: the generic agent keeps the whole catalog. Fixing the
	// generic's own pipeline is a separate effort (it is on the map's
	// Out of scope), so the cut must stop at the dispatched toolset.
	generic := useSkillTool(t, tools.GetSkillAndTools())
	enum := skillEnum(t, generic)
	for _, name := range append(slices.Clone(pipelineSkills), "mermaid") {
		if !slices.Contains(enum, name) {
			t.Errorf("the cut reached the generic agent, which lost %q: %v", name, enum)
		}
		if !strings.Contains(generic.Description.Value, name) {
			t.Errorf("the generic catalog no longer lists %q: %s", name, generic.Description.Value)
		}
	}
}

// TestReadSkillFileReachesMermaidWithoutTheCatalog is the second acceptance of
// the cut: with use_skill gone from the dispatched toolset, read_skill_file
// still serves the mermaid reference. The handler resolves against the full
// registry and checks no activation (ReadSkillFileNoActivation), so losing the
// catalog loses no reach.
func TestReadSkillFileReachesMermaidWithoutTheCatalog(t *testing.T) {
	t.Chdir(repoRoot(t))

	if findTool(tools.GetFileAndSkillTools(), skills.ToolReadSkillFile) == nil {
		t.Fatalf("%s is not offered to a dispatched Subagente", skills.ToolReadSkillFile)
	}

	call := responses.ResponseFunctionToolCall{
		CallID:    "call_read",
		Name:      skills.ToolReadSkillFile,
		Arguments: `{"skill_name": "mermaid", "relative_path": "references/flowchart.md"}`,
	}
	got := callOutput(t, (&llm2.Agent{}).SkillCall(context.Background(), call))
	if strings.HasPrefix(got, "erro:") {
		t.Fatalf("read_skill_file references/flowchart.md: %s", got)
	}
	if !strings.Contains(strings.ReplaceAll(got, "\r\n", "\n"), "```mermaid\nflowchart") {
		t.Errorf("read_skill_file returned no flowchart example: %s", got)
	}
}

// findTool returns the named function tool of a toolset, or nil when the
// toolset does not offer it. Unlike useSkillTool it does not fail the test:
// absence is the assertion for the dispatched toolset.
func findTool(toolset []responses.ToolUnionParam, name string) *responses.FunctionToolParam {
	for _, tool := range toolset {
		if fn := tool.OfFunction; fn != nil && fn.Name == name {
			return fn
		}
	}
	return nil
}

// toolNames lists the function names a toolset offers the model.
func toolNames(toolset []responses.ToolUnionParam) []string {
	names := make([]string, 0, len(toolset))
	for _, tool := range toolset {
		if tool.OfFunction != nil {
			names = append(names, tool.OfFunction.Name)
		}
	}
	return names
}

// useSkillEnum returns the skill names the use_skill tool offers the model.
func useSkillEnum(t *testing.T, toolset []responses.ToolUnionParam) []string {
	t.Helper()
	return skillEnum(t, useSkillTool(t, toolset))
}

// useSkillTool returns the use_skill tool of a toolset, failing the test when
// it is absent. Use findTool where absence is the expected outcome.
func useSkillTool(t *testing.T, toolset []responses.ToolUnionParam) *responses.FunctionToolParam {
	t.Helper()
	fn := findTool(toolset, skills.ToolUseSkill)
	if fn == nil {
		t.Fatal("use_skill tool not built")
	}
	return fn
}

// skillEnum returns the enum of the use_skill tool's skill_name parameter.
func skillEnum(t *testing.T, fn *responses.FunctionToolParam) []string {
	t.Helper()
	props, _ := fn.Parameters["properties"].(map[string]any)
	name, _ := props["skill_name"].(map[string]any)
	enum, ok := name["enum"].([]string)
	if !ok {
		t.Fatalf("use_skill has no string enum: %#v", fn.Parameters)
	}
	return enum
}
