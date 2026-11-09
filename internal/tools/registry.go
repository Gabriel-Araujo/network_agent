package tools

import (
	"context"
	"encoding/json"

	"github.com/Gabriel-Araujo/network_agent/internal/logger"
	"github.com/Gabriel-Araujo/network_agent/internal/skills"
	filetools "github.com/Gabriel-Araujo/network_agent/internal/tools/file"
	"github.com/openai/openai-go/v3/responses"
)

// log é o handle do pacote (uma declaração por pacote, não por arquivo).
var log = logger.Named("TOOLS")

const DEFAULT_ROOT_DIRECTORY = "."

var Tools = FileTools

// GetFileAndSkillTools returns the toolset of a dispatched Subagente: the file
// tools plus read_skill_file, and no use_skill — so no skill catalog reaches
// it. Dropping the catalog is the cut, not filtering it: every skill in the
// registry is either the generic's pipeline (frr-intent-analyzer, the
// frr-file-builder stub, frr-rag-retriever, which no live prompt even names)
// or "mermaid", already *mounted* in the prompt of the two Subagentes who draw
// (promptSkills). So the catalog held nothing a dispatched Subagente could
// use, while its descriptions ordered activation anyway: 12 of the 30 calls in
// the ticket 28 batch went to the two pipeline skills, and t11 spent both its
// calls there and never reached mermaid. read_skill_file stays, so the mermaid
// references remain reachable — the Illustrator and Teacher configs name both
// the tool and the skill, and HandleSkillCall resolves against the full
// registry.
func GetFileAndSkillTools() []responses.ToolUnionParam {
	return append([]responses.ToolUnionParam{skills.ReadSkillFileTool()}, Tools...)
}

// GetSkillAndTools is the generic agent's toolset: file + the whole skills
// catalog, plus the subagent tool.
func GetSkillAndTools() []responses.ToolUnionParam {
	skillsTools := skills.LoadSkills().BuildTools()

	return append(append(skillsTools, Tools...), SubAgentTool)
}

func CallFunction(
	ctx context.Context,
	call responses.ResponseFunctionToolCall,
) responses.ResponseInputItemUnionParam {
	log.Debugf("function '%s' called with arguments: [%s]\n", call.Name, call.Arguments)

	args := make(map[string]string)

	err := json.Unmarshal([]byte(call.Arguments), &args)
	if err != nil {
		log.Error("argumentos inválidos na chamada de função", "name", call.Name, "err", err)
		return responses.ResponseInputItemParamOfFunctionCallOutput("Error: Failed to get args from function call object.")
	}

	args["workingDirectory"] = DEFAULT_ROOT_DIRECTORY

	switch call.Name {
	case filetools.READ_TOOL_NAME:
		return responses.ResponseInputItemParamOfFunctionCallOutput(filetools.ReadFile(args["workingDirectory"], args["filePath"]))
	case filetools.EDIT_TOOL_NAME:
		return responses.ResponseInputItemParamOfFunctionCallOutput(filetools.EditFile(
			args["workingDirectory"],
			args["filePath"],
			args["oldText"],
			args["newText"]))
	case filetools.WRITE_TOOL_NAME:
		return responses.ResponseInputItemParamOfFunctionCallOutput(filetools.WriteFile(
			args["workingDirectory"],
			args["filePath"],
			args["content"]))
	case filetools.SEARCH_TOOL_NAME:
		return responses.ResponseInputItemParamOfFunctionCallOutput(filetools.SearchPattern(
			args["workingDirectory"],
			args["path"],
			args["pattern"]))

	default:
		return responses.ResponseInputItemParamOfFunctionCallOutput("Error: Called invalid function")
	}
}
