package agent

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Gabriel-Araujo/network_agent/internal/paths"
	"github.com/Gabriel-Araujo/network_agent/internal/skills"
	"github.com/Gabriel-Araujo/network_agent/internal/tools"
	filetools "github.com/Gabriel-Araujo/network_agent/internal/tools/file"
	"github.com/openai/openai-go/v3/responses"
)

// deniedTools is the table of tools a named Subagente is dispatched *without*,
// next to the name -> folder mapping it belongs with. The prompt is not a
// barrier: the Modifier's prompt forbids WriteFile and it still called it on a
// missing path, recreating the file from the transcript (ticket 13, K2). A name
// absent from the table gets the whole dispatched toolset.
var deniedTools = map[string][]string{
	"modifier": {filetools.WRITE_TOOL_NAME},
}

// promptSkills is the table of Skills *mounted* into a named Subagente's system
// prompt, next to deniedTools and in the same form. Mounted, not activated: the
// model never runs a consultative read — use_skill was 0/15 in the diagram
// batch and the Builder's ReadFile of the tech pack 0/4 — so the syntax has to
// be in the prompt before the turn starts instead of waiting on a call the
// model does not make (ticket 22). The Builder gets none: it does not draw.
var promptSkills = map[string][]string{
	"illustrator": {"mermaid"},
	"teacher":     {"mermaid"},
}

// ForSubagent returns the Agent that answers the turn as the named Subagente:
// same connection and model, the system prompt is
// resources/agents/<name>/AGENT.md mounted whole followed by whatever
// promptSkills mounts for this name, tools are the file tools plus
// read_skill_file, without the subagent tool (no nesting), without the
// use_skill catalog, and minus whatever deniedTools withholds from this name.
// A missing folder or AGENT.md — or an empty one — is an explicit error, never
// silent. The name -> folder table is the routing List in rag.Lists.
func (a *Agent) ForSubagent(name string) (*Agent, error) {
	root, err := paths.RepoFile("resources/agents")
	if err != nil {
		return nil, fmt.Errorf("locating the subagent configs root: %w", err)
	}
	return a.forSubagentAt(root, name)
}

// forSubagentAt is ForSubagent's testable seam: the subagent configs root comes
// from the caller (tests build a temporary directory).
func (a *Agent) forSubagentAt(root, name string) (*Agent, error) {
	if strings.TrimSpace(name) == "" {
		return nil, errors.New("subagent with no name")
	}
	dir := filepath.Join(root, name)
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("folder of subagent %q not found (%s)", name, dir)
	}
	file := filepath.Join(dir, "AGENT.md")
	raw, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("configuration of subagent %q not loaded (%s): %w", name, file, err)
	}
	if strings.TrimSpace(string(raw)) == "" {
		return nil, fmt.Errorf("configuration of subagent %q is empty (%s)", name, file)
	}
	prompt, err := mountPromptSkills(string(raw), name)
	if err != nil {
		return nil, err
	}
	return &Agent{
		Client:           a.Client,
		ModelName:        a.ModelName,
		providerURL:      a.providerURL,
		apiKey:           a.apiKey,
		modelContextSize: a.modelContextSize,
		availableTools:   subagentTools(name),
		systemPrompt:     prompt,
	}, nil
}

// mountPromptSkills appends the body of every Skill promptSkills names for this
// Subagente to its AGENT.md, raw and separated by a blank line: the SKILL.md
// already opens with its own heading, so a wrapper would duplicate it. A name
// absent from the registry is an explicit error, like a missing AGENT.md — the
// table is code, and a prompt silently without the syntax is the very failure
// the mounting fixes. A name absent from the table mounts nothing.
func mountPromptSkills(prompt, name string) (string, error) {
	mounted := promptSkills[name]
	if len(mounted) == 0 {
		return prompt, nil
	}
	registry := skills.LoadSkills()
	var out strings.Builder
	out.WriteString(prompt)
	for _, skillName := range mounted {
		skill, ok := registry.Get(skillName)
		if !ok {
			return "", fmt.Errorf("skill %q mounted in the prompt of subagent %q not found in the skills registry", skillName, name)
		}
		out.WriteString("\n\n")
		out.WriteString(skill.Body)
	}
	return out.String(), nil
}

// subagentTools is the dispatched toolset for the named Subagente: the file
// tools plus read_skill_file (see tools.GetFileAndSkillTools for why no
// use_skill), minus the tools deniedTools withholds from this name. The denied
// tool is gone from the request, so the model never reads its description; a
// call by name is refused by callTool.
func subagentTools(name string) []responses.ToolUnionParam {
	toolset := tools.GetFileAndSkillTools()
	denied := deniedTools[name]
	if len(denied) == 0 {
		return toolset
	}
	return slices.DeleteFunc(toolset, func(t responses.ToolUnionParam) bool {
		return t.OfFunction != nil && slices.Contains(denied, t.OfFunction.Name)
	})
}
