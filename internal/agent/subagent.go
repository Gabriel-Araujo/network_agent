package agent

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Gabriel-Araujo/network_agent/internal/config"
	"github.com/Gabriel-Araujo/network_agent/internal/tools"
	filetools "github.com/Gabriel-Araujo/network_agent/internal/tools/file"
	"github.com/openai/openai-go/v3/responses"
)

// deniedTools is the table of tools a named Subagente is dispatched *without*,
// next to the name -> folder mapping it belongs with. The prompt is not a
// barrier: the Modifier's prompt forbids WriteFile and it still called it on a
// missing path, recreating the file from the transcript (ticket 13, K2). A name
// absent from the table gets the whole dispatched toolset. The Teacher is in it
// because it is the default of the empty routing: every "nenhum" lands on it,
// so its "never writes or edits a file" has to be the toolset's rule and not
// the prompt's.
var deniedTools = map[string][]string{
	"modifier": {filetools.WRITE_TOOL_NAME},
	"teacher":  {filetools.WRITE_TOOL_NAME, filetools.EDIT_TOOL_NAME},
}

// subagentSkills is the table of Skills a named Subagente *activates* through
// use_skill, next to deniedTools and in the same form: the skills its AGENT.md
// names and orders activated before drawing. The activation tool is narrowed
// to exactly these names (tools.GetFileAndSkillTools), so no other description
// can order a different activation — the 12 of 30 wasted calls the ticket 28
// batch measured. Nothing is mounted in the prompt: the use_skill call is the
// only path the body travels (owner decision, ticket 35). A name absent from
// the registry is an explicit error. The Builder gets none: it does not draw.
var subagentSkills = map[string][]string{
	"illustrator": {"mermaid"},
	"teacher":     {"mermaid"},
}

// ForSubagent returns the Agent that answers the turn as the named Subagente:
// same connection and model, the system prompt is
// resources/agents/<name>/AGENT.md — under the Agent root, the mirror
// config.MaterialFile projects from the binary — mounted whole and nothing
// else, tools are the file tools plus read_skill_file and, for a name whose
// subagentSkills lists skills, use_skill narrowed to them, without the
// subagent tool (no nesting) and minus whatever deniedTools withholds from
// this name. A missing folder or AGENT.md — or an empty one — is an explicit
// error, never silent. The name -> folder table is the routing List in
// rag.Lists.
func (a *Agent) ForSubagent(name string) (*Agent, error) {
	root, err := config.MaterialFile("resources/agents")
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
	toolset, err := subagentTools(name)
	if err != nil {
		return nil, err
	}
	return &Agent{
		Client:           a.Client,
		ModelName:        a.ModelName,
		providerURL:      a.providerURL,
		apiKey:           a.apiKey,
		modelContextSize: a.modelContextSize,
		availableTools:   toolset,
		workspace:        a.workspace,
		systemPrompt:     string(raw),
	}, nil
}

// subagentTools is the dispatched toolset for the named Subagente: the file
// tools plus read_skill_file and — only for a name whose subagentSkills lists
// skills — use_skill narrowed to those skills (see tools.GetFileAndSkillTools),
// minus the tools deniedTools withholds from this name. The denied tool is
// gone from the request, so the model never reads its description; a call by
// name is refused by callTool.
func subagentTools(name string) ([]responses.ToolUnionParam, error) {
	toolset, err := tools.GetFileAndSkillTools(subagentSkills[name])
	if err != nil {
		return nil, err
	}
	denied := deniedTools[name]
	if len(denied) == 0 {
		return toolset, nil
	}
	return slices.DeleteFunc(toolset, func(t responses.ToolUnionParam) bool {
		return t.OfFunction != nil && slices.Contains(denied, t.OfFunction.Name)
	}), nil
}
