package file_tools

import "github.com/openai/openai-go/v3"

// READ TOOL CONSTANTS
var READ_MAX_CHARS_TO_READ = 10000
var READ_TOOL_NAME = "ReadFile"
var READ_TOOL_DESCRIPTION = "Read a file from the local filesystem."
var READ_TOOL_PARAMETERS = openai.FunctionParameters{
	"type": "object",
	"properties": map[string]any{
		"file_path": map[string]any{
			"type":        "string",
			"description": "File path, relative to the working directory",
		},
	},
	"required": []string{"file_path"},
}

// EDIT TOOL CONSTANTS
var MAX_EDIT_FILE_SIZE = 1024 * 1024 * 1024 // 1 GiB (stat bytes)
var EDIT_TOOL_NAME = "EditFile"
var EDIT_TOOL_DESCRIPTION = `Edit File content. Performs exact string replacements in files.
Points to pay attention:
- When editing text from Read tool output, ensure you preserve the exact indentation (tabs/spaces) as it appears AFTER the line number prefix. Everything after that is the actual file content to match. Never include any part of the line number prefix in the old_string or new_string;
- ALWAYS prefer editing existing files in the codebase. NEVER write new files unless explicitly required;
- Do NOT use emojis.`
var EDIT_TOOL_PARAMETERS = openai.FunctionParameters{
	"type": "object",
	"properties": map[string]any{
		"file_path": map[string]any{
			"type":        "string",
			"description": "File path, relative to the working directory",
		},
		"old_text": map[string]any{
			"type":        "string",
			"description": "Text existing in the file in the moment of first read.",
		},
		"new_text": map[string]any{
			"type":        "string",
			"description": "Text with the edited content of the file.",
		},
	},
	"required": []string{"file_path", "old_text", "new_text"},
}
