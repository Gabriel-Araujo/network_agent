package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"regexp"
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

	if enum := useSkillEnum(t, mustUseSkillTool(t, reg)); !slices.Contains(enum, "mermaid") {
		t.Errorf("use_skill does not enumerate mermaid: %v", enum)
	}

	a := &llm2.Agent{}

	// Activation is the only path the rules travel — nothing is mounted in the
	// prompt (subagentSkills, ticket 35) — so a body without the styling ban is
	// an Illustrator never taught the linkStyle form a measured production block
	// broke (ticket 28). The rule's name is the canary, not its wording.
	activated, _ := a.SkillCall(context.Background(), responses.ResponseFunctionToolCall{
		CallID:    "call_use",
		Name:      skills.ToolUseSkill,
		Arguments: `{"skill_name": "mermaid"}`,
	})
	switch body := callOutput(t, activated); {
	case strings.HasPrefix(body, "erro:"):
		t.Errorf("use_skill mermaid: %s", body)
	case !strings.Contains(body, "linkStyle"):
		t.Error("the activated mermaid body no longer bans linkStyle")
	}

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
		item, _ := a.SkillCall(context.Background(), call)
		got := callOutput(t, item)
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

// TestFrrFileBuilderSkillInCatalog pins the frr-file-builder material: the
// skill loads, use_skill delivers a body that is no longer the stub and that
// never names the file tools (writing and replying belong to the agent that
// activates it), and read_skill_file reaches every file of the bgp pack the
// body orders read — TECH.md, syntax.md, doctrine.md — plus every example the
// TECH.md index lists. The index is the only way the model finds an example,
// since read_skill_file cannot list a folder, so a row naming a missing file
// is an example the model asks for and never gets.
func TestFrrFileBuilderSkillInCatalog(t *testing.T) {
	t.Chdir(repoRoot(t))

	const skill = "frr-file-builder"
	if _, ok := skills.LoadSkills().Get(skill); !ok {
		t.Fatalf("skill %q not loaded from resources/agents/skills", skill)
	}

	a := &llm2.Agent{}
	read := func(path string) string {
		t.Helper()
		item, _ := a.SkillCall(context.Background(), responses.ResponseFunctionToolCall{
			CallID:    "call_read",
			Name:      skills.ToolReadSkillFile,
			Arguments: `{"skill_name": "` + skill + `", "relative_path": "` + path + `"}`,
		})
		got := callOutput(t, item)
		if strings.HasPrefix(got, "erro:") {
			t.Errorf("read_skill_file %s: %s", path, got)
		}
		return strings.ReplaceAll(got, "\r\n", "\n")
	}

	activated, _ := a.SkillCall(context.Background(), responses.ResponseFunctionToolCall{
		CallID:    "call_use",
		Name:      skills.ToolUseSkill,
		Arguments: `{"skill_name": "` + skill + `"}`,
	})
	body := callOutput(t, activated)
	switch {
	case strings.HasPrefix(body, "erro:"):
		t.Fatalf("use_skill %s: %s", skill, body)
	case strings.Contains(body, "reached frr-file-builder"):
		t.Error("the activated body is still the stub")
	case !strings.Contains(body, "<recovered_context>"):
		t.Error("the activated body no longer orders the RAG read")
	}
	for _, tool := range []string{"WriteFile", "EditFile"} {
		if strings.Contains(body, tool) {
			t.Errorf("the activated body names %s: the content is the skill's, the write is the agent's", tool)
		}
	}

	for _, path := range []string{"references/bgp/syntax.md", "references/bgp/doctrine.md"} {
		read(path)
	}
	index := read("references/bgp/TECH.md")
	examples := regexp.MustCompile("`(examples/[a-z0-9-]+\\.md)`").FindAllStringSubmatch(index, -1)
	if len(examples) == 0 {
		t.Fatal(`TECH.md of the bgp pack lists no example under "Examples"`)
	}
	for _, m := range examples {
		if got := read("references/bgp/" + m[1]); !strings.Contains(got, "```") {
			t.Errorf("%s has no fenced frr.conf shape", m[1])
		}
	}
}

// pipelineSkills are the skills a dispatched Subagente has no use for: the
// frr-file-builder, whose consumers are the Builder and the Modifier, the
// generic's intent analyzer, and the RAG retriever
// whose briefing input no live prompt produces any more (the turn's RAG is Go,
// internal/turn). Their descriptions order activation on any FRR question, so
// a dispatched Subagente that can read the catalog spends calls on them: 12 of
// 30 in the ticket 28 batch, and t11 never reached mermaid.
var pipelineSkills = []string{"frr-file-builder", "frr-intent-analyzer", "frr-rag-retriever"}

// TestSkillActivationNarrowedForTheSubagente pins the shape the owner chose
// (ticket 35): a dispatched Subagente carries use_skill narrowed to the skills
// its own prompt names — the catalog the model reads can only order the
// activation of those — plus read_skill_file and the file tools. A name with
// no skills carries no activation tool at all, and an unknown
// name is an explicit error. What must NOT return is the full catalog: the
// three pipeline skills' descriptions ordered activation on any FRR question —
// 12 of 30 calls in the ticket 28 batch, and t11 never reached mermaid — so
// their names must not reach a dispatched model through any tool.
func TestSkillActivationNarrowedForTheSubagente(t *testing.T) {
	t.Chdir(repoRoot(t))

	dispatched, err := tools.GetFileAndSkillTools([]string{"mermaid"})
	if err != nil {
		t.Fatalf("building the narrowed toolset: %v", err)
	}

	// Half one: use_skill exists, narrowed to exactly the named skill.
	fn := findTool(dispatched, skills.ToolUseSkill)
	if fn == nil {
		t.Fatalf("the dispatched Subagente lost %s: %v", skills.ToolUseSkill, toolNames(dispatched))
	}
	if enum := skillEnum(t, fn); !slices.Equal(enum, []string{"mermaid"}) {
		t.Errorf("dispatched %s enum = %v, want exactly [mermaid]", skills.ToolUseSkill, enum)
	}

	// What must survive alongside it: read_skill_file is how the Illustrator
	// and Teacher configs reach references/flowchart.md and its two siblings.
	if findTool(dispatched, skills.ToolReadSkillFile) == nil {
		t.Errorf("the narrowing took %s with it: %v", skills.ToolReadSkillFile, toolNames(dispatched))
	}

	// No pipeline skill name reaches the dispatched model through any tool:
	// the catalog text was the carrier of the hijack, and this is the half
	// ticket 18 checked on the description.
	blob, err := json.Marshal(dispatched)
	if err != nil {
		t.Fatalf("marshaling the dispatched toolset: %v", err)
	}
	for _, name := range pipelineSkills {
		if bytes.Contains(blob, []byte(name)) {
			t.Errorf("the dispatched toolset still names the skill %q: %s", name, blob)
		}
	}

	// A name with no skills gets no activation tool at all: the empty enum
	// would be a lie, and a description no model can act on.
	plain, err := tools.GetFileAndSkillTools(nil)
	if err != nil {
		t.Fatalf("building the skill-less toolset: %v", err)
	}
	if fn := findTool(plain, skills.ToolUseSkill); fn != nil {
		t.Errorf("the skill-less toolset offers %s: %v", skills.ToolUseSkill, skillEnum(t, fn))
	}

	// An unknown name is an explicit error: the list is code.
	if _, err := tools.GetFileAndSkillTools([]string{"no-such-skill"}); err == nil {
		t.Error("GetFileAndSkillTools accepted a skill name the registry does not have")
	}
}

// TestReadSkillFileServesTheReferenceWithoutActivation: read_skill_file serves
// the mermaid reference regardless of activation. The handler resolves against
// the full registry and checks no activation (ReadSkillFileNoActivation), so a
// turn that never called use_skill still reaches the references.
func TestReadSkillFileServesTheReferenceWithoutActivation(t *testing.T) {
	t.Chdir(repoRoot(t))

	toolset, err := tools.GetFileAndSkillTools([]string{"mermaid"})
	if err != nil {
		t.Fatalf("building the narrowed toolset: %v", err)
	}
	if findTool(toolset, skills.ToolReadSkillFile) == nil {
		t.Fatalf("%s is not offered to a dispatched Subagente", skills.ToolReadSkillFile)
	}

	call := responses.ResponseFunctionToolCall{
		CallID:    "call_read",
		Name:      skills.ToolReadSkillFile,
		Arguments: `{"skill_name": "mermaid", "relative_path": "references/flowchart.md"}`,
	}
	item, _ := (&llm2.Agent{}).SkillCall(context.Background(), call)
	got := callOutput(t, item)
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

func mustUseSkillTool(t *testing.T, reg *skills.Registry) []responses.ToolUnionParam {
	t.Helper()
	tool, err := reg.UseSkillTool([]string{"mermaid"})
	if err != nil {
		t.Fatalf("building the use_skill tool: %v", err)
	}
	return []responses.ToolUnionParam{tool}
}
