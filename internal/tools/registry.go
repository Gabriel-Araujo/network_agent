package tools

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/Gabriel-Araujo/network_agent/internal/skills"
	filetools "github.com/Gabriel-Araujo/network_agent/internal/tools/file"
	"github.com/openai/openai-go/v3"
)

var DEFAULT_ROOT_DIRECTORY = "."
var Tools = []openai.ChatCompletionToolUnionParam{FileReadTool, FileEditTool, FileWriteTool, SearchFileTool}

func GetSkillAndTools() []openai.ChatCompletionToolUnionParam {
	reg := skills.LoadSkills()
	skillsTools := reg.BuildTools()

	return append(skillsTools, Tools...)
}

func CallFunction(functionCall openai.ChatCompletionChunkChoiceDeltaToolCall, verbose bool) openai.ChatCompletionMessageParamUnion {
	if verbose {
		log.Printf("function '%s' called with arguments: [%s]\n", functionCall.Function.Name, functionCall.Function.Arguments)
	}

	args := make(map[string]string)

	err := json.Unmarshal([]byte(functionCall.Function.Arguments), &args)
	if err != nil {
		log.Default().Println(err)
		return openai.ToolMessage("Error: Failed to get args from function call object.", functionCall.ID)
	}

	args["workingDirectory"] = DEFAULT_ROOT_DIRECTORY

	switch functionCall.Function.Name {
	case filetools.READ_TOOL_NAME:
		return openai.ToolMessage(
			filetools.ReadFile(args["workingDirectory"], args["filePath"]),
			functionCall.ID)
	case filetools.EDIT_TOOL_NAME:
		return openai.ToolMessage(filetools.EditFile(
			args["workingDirectory"],
			args["filePath"],
			args["oldText"],
			args["newText"]),
			functionCall.ID)
	case filetools.WRITE_TOOL_NAME:
		return openai.ToolMessage(filetools.WriteFile(
			args["workingDirectory"],
			args["filePath"],
			args["content"]),
			functionCall.ID)
	case filetools.SEARCH_TOOL_NAME:
		return openai.ToolMessage(filetools.SearchPattern(
			args["workingDirectory"],
			args["path"],
			args["pattern"]),
			functionCall.ID)
	case skills.ToolUseSkill:
		reg := skills.LoadSkills()
		var args struct {
			SkillName string `json:"skill_name"`
		}
		if err := json.Unmarshal([]byte(functionCall.Function.Arguments), &args); err != nil {
			return openai.ToolMessage(fmt.Sprintf("erro: argumentos inválidos: %v", err), functionCall.ID)
		}
		sk, ok := reg.Get(args.SkillName)
		if !ok {
			return openai.ToolMessage(fmt.Sprintf("erro: skill %q não encontrada", args.SkillName), functionCall.ID)
		}
		return openai.ToolMessage(sk.Body, functionCall.ID)
	case skills.ToolReadSkillFile:
		reg := skills.LoadSkills()

		var args struct {
			SkillName    string `json:"skill_name"`
			RelativePath string `json:"relative_path"`
		}
		if err := json.Unmarshal([]byte(functionCall.Function.Arguments), &args); err != nil {
			return openai.ToolMessage(fmt.Sprintf("erro: argumentos inválidos: %v", err), functionCall.ID)
		}
		content, err := reg.ReadFile(args.SkillName, args.RelativePath)
		if err != nil {
			return openai.ToolMessage(fmt.Sprintf("erro: %v", err), functionCall.ID)
		}
		return openai.ToolMessage(content, functionCall.ID)

	default:
		return openai.ToolMessage("Error: Called invalid function", functionCall.ID)
	}
}
