package filetools

import (
	"github.com/Gabriel-Araujo/network_agent/internal/logger"
	"github.com/openai/openai-go/v3"
)

// log é o handle do pacote (uma declaração por pacote, não por arquivo).
var log = logger.Named("TOOLS")

// AS DUAS RAÍZES
//
// Depois do ticket 10 não existe "o diretório de trabalho": um caminho resolve
// contra uma de duas raízes, e o prefixo diz qual. Os três avisos abaixo são o
// que o modelo lê quando a raiz não serve, e são distintos de propósito —
// o conserto de cada caso é diferente (escolher pasta × reconectar o drive ×
// escrever noutro lugar).
const (
	// MIRROR_PREFIX manda um caminho para o espelho de material do agente em
	// vez do Workspace. É literalmente "resources/" porque a projeção preserva
	// o caminho embutido (ADR 0004): o "resources/agents/teacher/tech-packs/"
	// que o AGENT.md do Teacher ensina resolve contra o home sem uma linha de
	// prompt mudar.
	MIRROR_PREFIX = "resources/"

	// NO_WORKSPACE_NOTICE is the result when a path needs the user's folder and
	// no folder has been chosen. It is not a transient error: retrying cannot
	// fix it, and only the user can.
	NO_WORKSPACE_NOTICE = "Error: no folder has been chosen for the user's files, so there is nowhere to read or write them. Nothing was read, written or created. Tell the user to choose a folder in the app's Settings. Do not retry, and do not invent a path."

	// WORKSPACE_GONE_NOTICE is the result when a folder was chosen and is no
	// longer there — renamed, unmounted, on a drive that was pulled. Told apart
	// from NO_WORKSPACE_NOTICE because the fix differs; the folder is appended.
	WORKSPACE_GONE_NOTICE = "Error: the folder chosen for the user's files is not there anymore, so nothing was read, written or created. Tell the user the folder is missing and that the drive holding it may need to be reconnected. Do not retry, and do not write anywhere else. The folder is: "

	// READ_ONLY_ROOT_NOTICE is the result when a write or an edit resolves
	// inside the app's own directory — the agent material under the mirror
	// prefix, or the app data beside it. That root is read-only: the material
	// is the same for every user and nothing in a turn is allowed to change it.
	// The path asked for is appended.
	READ_ONLY_ROOT_NOTICE = "Error: that path is inside the agent's own read-only material, which is never written, edited or replaced. Nothing was changed. Write the user's files under the folder they chose instead. The path refused is: "
)

// TWO_ROOTS_RULE is the path contract every file tool shares, mounted into all
// four descriptions so they cannot teach different roots. It is written for the
// model: which prefix goes where, and what it may not conclude from a refusal.
const TWO_ROOTS_RULE = `
## Path Resolution: two roots

A path is not resolved against one working directory. There are two roots, and the prefix of the path picks which one:

1. A path starting with ` + "`" + MIRROR_PREFIX + "`" + ` resolves against the agent's own material: your support files, the same for every user. This root is READ-ONLY. Reading and searching it works; writing or editing anything under it is refused and changes nothing.
2. Every other path resolves against the folder the user chose for their files. It is the only root you can write to.

Rules:

- Never try to reach one root by prefixing or climbing out of the other. A path that resolves outside its root is refused; it is not an invitation to retry with ` + "`" + `../` + "`" + `.
- The results carry absolute paths. Reuse the absolute path a tool returned when you refer to the same file again, and show it to the user as-is: with two roots, a bare relative path does not say which root it came from.
- If no folder has been chosen, or the chosen folder is gone, paths of kind 2 are refused and nothing happens. Report that to the user; never claim a file was read, written or created after a refusal, and never fill in its content from this conversation.`

// READ TOOL CONSTANTS
const (
	READ_MAX_CHARS_TO_READ = 64 * 1024 // 64 KiB (bytes); larger files come back truncated
	READ_TOOL_NAME         = "ReadFile"
	// READ_NOT_FOUND_NOTICE is the result when the target file does not exist,
	// told apart from every other read failure; the path is appended to it, and
	// the description below shows it to the model.
	READ_NOT_FOUND_NOTICE = "Error: File not found. Nothing was read, and this tool never creates files. Do not say the file was written, created, edited or updated, and do not show configuration from the conversation as the content of this file. Tell the user the file does not exist: "
	READ_TOOL_DESCRIPTION = `
Reads and returns the content of a file from the local filesystem.
` + TWO_ROOTS_RULE + `
## Workflow & Logic:
### 1. Locating the file (Mandatory Step):

Before attempting to read a file, inspect the provided argument.
If the argument is only a filename (e.g., frr.conf) that does not say which directory it is in, you must first call SearchFile to locate it, and read it by the absolute path the search returned.
Only proceed to reading the file once the path is confirmed.

### 2. Content Retrieval:

Once the path is identified, retrieve the raw text content of the file.

### Error Handling:

- Path resolving outside its root: Error: Filepath outside the permitted working directory
- No folder chosen for the user's files: ` + NO_WORKSPACE_NOTICE + `
- The chosen folder is gone: ` + WORKSPACE_GONE_NOTICE + `<folder>
- Target file does not exist: ` + READ_NOT_FOUND_NOTICE + `<path>
- Any other read failure (no permission, path is a directory): Error: failed to read File content
- Content over the cap comes back cut, closed by a truncation marker.`
)

var READ_TOOL_PARAMETERS = openai.FunctionParameters{
	"type": "object",
	"properties": map[string]any{
		"filePath": map[string]any{
			"type":        "string",
			"description": "Path of the file. Resolved against the folder the user chose, unless it starts with the agent-material prefix; an absolute path a tool already returned is accepted as-is.",
		},
	},
	"required": []string{"filePath"},
}

// EDIT TOOL CONSTANTS
const (
	MAX_EDIT_FILE_SIZE = 1024 * 1024 * 1024 // 1 GiB (stat bytes)
	EDIT_TOOL_NAME     = "EditFile"
	// EDIT_MULTIPLE_MATCHES_NOTICE opens the error when old_text matches more
	// than once; the occurrence count is appended to it, and the description
	// below shows it to the model.
	EDIT_MULTIPLE_MATCHES_NOTICE = "Error: old_text is not unique in the file. Nothing was changed. Include more surrounding lines in old_text so it matches exactly once. Occurrences found: "
	EDIT_TOOL_DESCRIPTION        = `Performs exact substring replacement within a specific file on the local filesystem. This tool is used to modify existing configuration files.
` + TWO_ROOTS_RULE + `
## Workflow & Logic:
### Mandatory Pre-conditions:

1. Read-Before-Write: Call ReadFile on the target file before EditFile, so old_string is copied from the current content of the file. The tool keeps no state between calls: it reads the file at call time and matches old_string against what is on disk.
2. File Existence: Always prioritize modifying existing files. Do not attempt to use EditFile to create a new file; use WriteFile (or the designated creation tool) for new files.

### Operational Rules for String Matching:

1. Exact Character Matching: The old_string must be a 100% literal, character-for-character match of the text found in the file.
2. Indentation & Whitespace Integrity:
2.1. When identifying the old_string from a ReadFile output that includes line numbers (e.g., 1:    interface eth0), you must ignore the line number and its separator (the prefix).
2.2. The old_string must begin immediately after the prefix, capturing all original indentation (tabs or spaces) exactly as they appear in the file.
2.3. Example: If ReadFile returns 10:  router bgp 65001, the old_string must be "  router bgp 65001" (including the two leading spaces), not "10:  router bgp 65001".
3. Uniqueness (enforced): the old_string must match exactly once in the file, and that single occurrence is replaced. If it matches more than once, the file is left untouched and the call fails. In repetitive configuration files, include the surrounding lines that make the target instance unique (for example, the neighbor line above the attribute being changed), instead of a short string that appears under every neighbor.

### Constraints:

- No New Files: Never use EditFile for file creation.
- No Emojis: Do not use emojis in any part of the tool call or subsequent responses.
- Integrity: Do not attempt to "clean up" or reformat the file while editing unless specifically instructed to do so; maintain the existing file structure and encoding.

### Error Handling:

Any result other than "Success: File edited" means the file on disk was not changed.

- Path resolving outside its root: Error Filepath outside the permitted working directory
- Path inside the agent's read-only material: ` + READ_ONLY_ROOT_NOTICE + `<path>
- No folder chosen for the user's files: ` + NO_WORKSPACE_NOTICE + `
- The chosen folder is gone: ` + WORKSPACE_GONE_NOTICE + `<folder>
- Target file does not exist: Error: File not found.
- old_string absent from the file: Error: old_text not found in file
- old_string matching more than once: ` + EDIT_MULTIPLE_MATCHES_NOTICE + `<occurrences>
- File modified between the read and the write: Error: File was modified by another process since it was last accessed.
- Checking the file for concurrent modification failed: Error: failed to check file staleness
- Reading the file failed: Error: failed to read File content
- Writing the file failed: Error: failed to write File content`
)

var EDIT_TOOL_PARAMETERS = openai.FunctionParameters{
	"type": "object",
	"properties": map[string]any{
		"filePath": map[string]any{
			"type":        "string",
			"description": "Path of the file. Resolved against the folder the user chose, unless it starts with the agent-material prefix; an absolute path a tool already returned is accepted as-is.",
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
	WRITE_TOOL_NAME = "WriteFile"
	// WRITE_REPLACED_NOTICE opens the result when WriteFile replaced an
	// existing file; the description below explains it to the model.
	WRITE_REPLACED_NOTICE  = "Replaced existing file: "
	WRITE_TOOL_DESCRIPTION = `Writes content to a specified path on the local filesystem. This tool is primarily used for creating new configuration files or performing full-file replacements.
` + TWO_ROOTS_RULE + `
## Workflow & Logic:
### Operational Logic:

1. Recursive Directory Creation:

- If the provided path includes directories that do not currently exist (e.g., configs/r1/frr.conf), the tool must attempt to create the entire directory tree required to house the target file (equivalent to mkdir -p).

2. Overwrite Behavior:

- If a file already exists at the destination path, the WriteFile tool will overwrite it completely. The existing content will be discarded and replaced by the new content provided in the tool call. There is no "append" functionality.
- The result tells the two cases apart. A new file returns the content written. A replaced file returns the line "` + WRITE_REPLACED_NOTICE + `<path>" followed by the content written; when you get it, tell the user that the existing file was replaced.

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

- If the path resolves outside its root: Error Filepath outside the permitted working directory
- If the path is inside the agent's read-only material: ` + READ_ONLY_ROOT_NOTICE + `<path>
- If no folder has been chosen for the user's files: ` + NO_WORKSPACE_NOTICE + `
- If the chosen folder is gone: ` + WORKSPACE_GONE_NOTICE + `<folder>
- If directory creation fails: Error: failed to create directory
- If writing the file fails: Error: failed to write File content`
)

var WRITE_TOOL_PARAMETERS = openai.FunctionParameters{
	"type": "object",
	"properties": map[string]any{
		"filePath": map[string]any{
			"type":        "string",
			"description": "Path of the file. Resolved against the folder the user chose, unless it starts with the agent-material prefix; an absolute path a tool already returned is accepted as-is.",
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
` + TWO_ROOTS_RULE + `
## Workflow & Logic:
### Operational Logic:

1. Search Scope (path):
- The search is confined to the directory specified in the path argument and its subdirectories.
- If no path is given consider it be ".", which is the root of the folder the user chose.
- The search starts at the provided path and descends recursively if the pattern dictates.

2. Pattern Matching (pattern):

2.1. Supports standard globbing syntax to filter results:
- *: Matches any number of characters within a single directory level (e.g., *.conf matches zebra.conf but not subdir/zebra.conf).
- **: Performs a recursive search through all subdirectories (e.g., **/*.cfg matches any .cfg file in the entire directory tree).
- ?: Matches exactly one character (e.g., router?.conf matches router1.conf).
2.2. Example Use Cases:
- To find all BGP configurations in a specific directory: SearchFiles(path="configs", pattern="**/*bgp*.conf")
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
			"description": "Root path to search in, resolved against the folder the user chose, unless it starts with the agent-material prefix. Defaults to \".\", the root of that folder.",
		},
	},
	"required": []string{"pattern", "path"},
}
