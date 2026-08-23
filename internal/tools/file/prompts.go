package filetools

import (
	"github.com/Gabriel-Araujo/network_agent/internal/logger"
	"github.com/openai/openai-go/v3"
)

// log é o handle do pacote (uma declaração por pacote, não por arquivo).
var log = logger.Named("TOOLS")

// READ TOOL CONSTANTS
const (
	READ_MAX_CHARS_TO_READ = 10000
	READ_TOOL_NAME         = "ReadFile"
	READ_TOOL_DESCRIPTION  = `
Reads and returns the content of a file from the local filesystem.
## Workflow & Logic:
### 1. Path Resolution (Mandatory Step):

Before attempting to read a file, inspect the provided argument.
If the argument is only a filename (e.g., frr.conf) or a relative path that does not resolve to a specific location, you must first call SearchFile(".", path, filename) to locate the absolute system path.
Only proceed to reading the file once the absolute path is confirmed.

### 2. Content Retrieval:

Once the absolute path is identified, retrieve the raw text content of the file.`
)

var READ_TOOL_PARAMETERS = openai.FunctionParameters{
	"type": "object",
	"properties": map[string]any{
		"filePath": map[string]any{
			"type":        "string",
			"description": "File path, relative to the working directory",
		},
	},
	"required": []string{"filePath"},
}

// EDIT TOOL CONSTANTS
const (
	MAX_EDIT_FILE_SIZE    = 1024 * 1024 * 1024 // 1 GiB (stat bytes)
	EDIT_TOOL_NAME        = "EditFile"
	EDIT_TOOL_DESCRIPTION = `Performs exact substring replacement within a specific file on the local filesystem. This tool is used to modify existing configuration files.

## Workflow & Logic:
### Mandatory Pre-conditions:

1. Read-Before-Write: You must successfully execute the ReadFile tool on the target file within the current conversation before calling EditFile. The system will reject any edit attempt if the file content has not been loaded into the context via ReadFile.
2. File Existence: Always prioritize modifying existing files. Do not attempt to use EditFile to create a new file; use WriteFile (or the designated creation tool) for new files.

### Operational Rules for String Matching:

1. Exact Character Matching: The old_string must be a 100% literal, character-for-character match of the text found in the file.
2. Indentation & Whitespace Integrity:
2.1. When identifying the old_string from a ReadFile output that includes line numbers (e.g., 1:    interface eth0), you must ignore the line number and its separator (the prefix).
2.2. The old_string must begin immediately after the prefix, capturing all original indentation (tabs or spaces) exactly as they appear in the file.
2.3. Example: If ReadFile returns 10:  router bgp 65001, the old_string must be "  router bgp 65001" (including the two leading spaces), not "10:  router bgp 65001".
3. Specificity: To avoid unintended side effects, ensure the old_string is as specific as possible to target the correct instance of the text, especially in repetitive configuration files.

### Constraints:

- No New Files: Never use EditFile for file creation.
- No Emojis: Do not use emojis in any part of the tool call or subsequent responses.
- Integrity: Do not attempt to "clean up" or reformat the file while editing unless specifically instructed to do so; maintain the existing file structure and encoding.

### Error Handling:

- If the old_string is not found in the file, return: Error: Substring match failed. The provided 'old_string' does not exist in the file content. Please re-read the file and verify exact characters/indentation.`
)

var EDIT_TOOL_PARAMETERS = openai.FunctionParameters{
	"type": "object",
	"properties": map[string]any{
		"filePath": map[string]any{
			"type":        "string",
			"description": "File path, relative to the working directory",
		},
		"oldText": map[string]any{
			"type":        "string",
			"description": "Text existing in the file in the moment of first read.",
		},
		"newText": map[string]any{
			"type":        "string",
			"description": "Text with the edited content of the file.",
		},
	},
	"required": []string{"filePath", "oldText", "newText"},
}

// Write TOOL CONSTANTS
const (
	WRITE_TOOL_NAME        = "WriteFile"
	WRITE_TOOL_DESCRIPTION = `Writes content to a specified path on the local filesystem. This tool is primarily used for creating new configuration files or performing full-file replacements.
## Workflow & Logic:
### Operational Logic:

1. Recursive Directory Creation:

- If the provided path includes directories that do not currently exist (e.g., /etc/frr/vty/config.conf), the tool must attempt to create the entire directory tree required to house the target file (equivalent to mkdir -p).

2. Overwrite Behavior:

- If a file already exists at the destination path, the WriteFile tool will overwrite it completely. The existing content will be discarded and replaced by the new content provided in the tool call. There is no "append" functionality.

3. Usage Policy (Strict Selection):

3.1. Use WriteFile ONLY when:
- The file does not exist (New File Creation).
- The entire content of the existing file must be replaced (Total Rewrite).

3.2 Use EditFile when:
- You are modifying, adding, or deleting specific lines or blocks within an existing file.

4. Note: Using WriteFile to perform minor edits to existing files is inefficient and is prohibited in this workflow.

### Constraints:

- No Emojis: Do not use emojis in any part of the tool call, parameters, or subsequent responses.
- Data Integrity: Ensure the content being written is valid and complete, as there is no way to "undo" an overwrite once the command is executed.

### Error Handling:

- If directory creation fails due to permissions or invalid paths: Error: Failed to create directory structure for path [path].
- If writing to a protected system area fails: Error: Permission denied when attempting to write to [path].
- If the provided path is syntactically invalid: Error: The provided path [path] is invalid.`
)

var WRITE_TOOL_PARAMETERS = openai.FunctionParameters{
	"type": "object",
	"properties": map[string]any{
		"filePath": map[string]any{
			"type":        "string",
			"description": "File path, relative to the working directory",
		},
		"content": map[string]any{
			"type":        "string",
			"description": "File content",
		},
	},
	"required": []string{"filePath", "content"},
}

const (
	SEARCH_TOOL_NAME        = "SearchFile"
	SEARCH_TOOL_DESCRIPTION = `Locates files within the filesystem using glob patterns (ex: **/*.go). This tool is essential for discovering file paths when only a filename or a partial directory structure is known.

## Workflow & Logic:
### Operational Logic:

1. Search Scope (path):
- The search is confined to the directory specified in the path argument and its subdirectories.
- If no path is given consider it be ".".
- The search starts at the provided path and descends recursively if the pattern dictates.

2. Pattern Matching (pattern):

2.1. Supports standard globbing syntax to filter results:
- *: Matches any number of characters within a single directory level (e.g., *.conf matches zebra.conf but not subdir/zebra.conf).
- **: Performs a recursive search through all subdirectories (e.g., **/*.cfg matches any .cfg file in the entire directory tree).
- ?: Matches exactly one character (e.g., router?.conf matches router1.conf).
2.2. Example Use Cases:
- To find all BGP configurations in a specific directory: SearchFiles(path="/etc/frr", pattern="**/*bgp*.conf")
- To find any YAML file in the current directory: SearchFiles(path=".", pattern="*.yaml")

### Constraints:
- Security: DO NOT try searching from the filesystem root (/), because you do not have access to it.
- No Emojis: Do not use emojis in any part of the tool call or subsequent responses.
- Output: The tool returns a list of absolute paths matching the pattern.
`
)

var SEARCH_TOOL_PARAMETERS = openai.FunctionParameters{
	"type": "object",
	"properties": map[string]any{
		"pattern": map[string]any{
			"type":        "string",
			"description": "Glob pattern to match files",
		},
		"path": map[string]any{
			"type":        "string",
			"description": "Root path to search in, relative to the working directory. Defaults to the current working directory.",
		},
	},
	"required": []string{"pattern", "path"},
}
