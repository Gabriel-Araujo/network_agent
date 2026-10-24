package classifier

import (
	"fmt"
	"strings"
)

// systemPrompt defines the classifier identity: it does not chat; it chooses.
const systemPrompt = `# Identity
You are a text classifier for a network engineering assistant (FRRouting). You do not chat: you read the user's request and choose values.

# Rules
- For EACH list below, choose EXACTLY ONE of its allowed values, spelled exactly as listed.
- If nothing in a list fits the request, use "" (empty string). "Nothing fits" is a valid answer; an invented value is never acceptable.
- Reply with a single JSON object only: one property per list and nothing else. No preamble, no code fences.`

// ragModePrompt enables the classifier's RAG behavior. The flag is supplied by
// the caller and changes internal behavior by feeding RAG search filters.
const ragModePrompt = `
# RAG Mode
This classification feeds RAG search filters. Prefer the specific value over "" whenever the request touches that protocol or daemon at all: the value scopes the document search for the whole request.`

// buildPrompt builds the system prompt from the lists and their values.
func buildPrompt(lists []List, ragMode bool) string {
	var b strings.Builder
	b.WriteString(systemPrompt)
	if ragMode {
		b.WriteString(ragModePrompt)
	}
	b.WriteString("\n\n# Lists\n")
	for _, list := range lists {
		fmt.Fprintf(&b, "\n## %s\n", strings.TrimSpace(list.Name))
		for _, value := range list.Values {
			fmt.Fprintf(&b, "- %q", value.Name)
			if description := strings.TrimSpace(value.Description); description != "" {
				fmt.Fprintf(&b, ": %s", description)
			}
			b.WriteString("\n")
		}
	}
	return b.String()
}

// buildCorrection includes a rejected-attempt error in the next request.
func buildCorrection(err error) string {
	return fmt.Sprintf(`Your previous answer was rejected: %v

Answer again with a single JSON object only, one property per list, each value spelled exactly as one of that list's allowed values — or "" when nothing fits. Never invent a value.`, err)
}

// buildSchema asks the provider for one string property per list, using the
// allowed values plus an explicit empty value.
func buildSchema(lists []List) map[string]any {
	props := make(map[string]any, len(lists))
	required := make([]string, 0, len(lists))
	for _, list := range lists {
		name := strings.TrimSpace(list.Name)
		enum := []string{""}
		for _, value := range list.Values {
			enum = append(enum, value.Name)
		}
		props[name] = map[string]any{"type": "string", "enum": enum}
		required = append(required, name)
	}
	return map[string]any{
		"type":                 "object",
		"properties":           props,
		"required":             required,
		"additionalProperties": false,
	}
}
