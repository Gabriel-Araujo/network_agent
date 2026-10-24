// Package classifier classifies an input by choosing exactly one value from each
// supplied List, or "" when no value matches. The complete contract is defined
// in .agent/classifier/issues/02-contrato-do-classificador.md.
package classifier

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	llm "github.com/Gabriel-Araujo/network_agent/internal/agent"
)

// Value is a candidate value in a List. Name is the canonical value: it is the
// only value that may be returned by classification, always with this spelling.
// Description is optional and helps the model distinguish similar values.
type Value struct {
	Name        string
	Description string
}

// List is a closed set of values for a dimension supplied by the caller.
// Classification returns one canonical Name from the Values, or "" when no
// value matches.
type List struct {
	Name   string
	Values []Value
}

// maxAttempts allows one call plus two correction retries. Invalid output is
// retried with the error included in the next request; no invented value leaks.
const maxAttempts = 3

// completeFunc returns the provider's raw response. It is the test seam that
// allows tests to inject canned responses without network access.
type completeFunc func(ctx context.Context, msgs []message, schema map[string]any) (string, error)

// message is one prompt turn in the format accepted by the provider.
type message struct {
	Role    string // "system" | "user" | "assistant"
	Content string
}

// Do classifies input, returning one canonical value per list or "" when no
// value matches. ragMode enables the classifier's RAG behavior.
func Do(ctx context.Context, input string, lists []List, ragMode bool, agent *llm.Agent) (map[string]string, error) {
	if agent == nil {
		return nil, errors.New("classifier: nil agent")
	}
	return classify(ctx, input, lists, ragMode, completeWithAgent(agent))
}

// classify runs the complete cycle: validate lists, build the prompt and
// schema, call the model, and validate its response. Providers may silently
// ignore the schema, so every path is checked against the allowed lists.
func classify(ctx context.Context, input string, lists []List, ragMode bool, complete completeFunc) (map[string]string, error) {
	index, err := validateLists(lists)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(input) == "" {
		return nil, errors.New("classifier: empty input")
	}

	schema := buildSchema(lists)
	msgs := []message{
		{Role: "system", Content: buildPrompt(lists, ragMode)},
		{Role: "user", Content: input},
	}

	var lastError error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		raw, err := complete(ctx, msgs, schema)
		if err != nil {
			lastError = fmt.Errorf("model call (attempt %d): %w", attempt, err)
			continue
		}
		values, err := validateResponse(raw, lists, index)
		if err == nil {
			return values, nil
		}
		lastError = fmt.Errorf("rejected response (attempt %d): %w", attempt, err)
		msgs = append(msgs,
			message{Role: "assistant", Content: raw},
			message{Role: "user", Content: buildCorrection(err)},
		)
	}
	return nil, fmt.Errorf("classifier: failed after %d attempts: %w", maxAttempts, lastError)
}

// listIndex resolves a list's canonical value from normalized spelling.
type listIndex map[string]string // normalized -> canonical Name

// validateLists validates the call before contacting the model and builds the
// resolution index. Equivalent values in one list ("BGP" and "bgp") are
// rejected because they would make resolution ambiguous.
func validateLists(lists []List) (map[string]listIndex, error) {
	if len(lists) == 0 {
		return nil, errors.New("classifier: no lists supplied")
	}
	index := make(map[string]listIndex, len(lists))
	for _, list := range lists {
		name := strings.TrimSpace(list.Name)
		if name == "" {
			return nil, errors.New("classifier: list has no name")
		}
		if _, duplicate := index[name]; duplicate {
			return nil, fmt.Errorf("classifier: duplicate list %q", name)
		}
		if len(list.Values) == 0 {
			return nil, fmt.Errorf("classifier: list %q has no values", name)
		}
		normalizedValues := make(listIndex, len(list.Values))
		for _, value := range list.Values {
			if strings.TrimSpace(value.Name) == "" {
				return nil, fmt.Errorf("classifier: list %q has an unnamed value", name)
			}
			normalized := normalize(value.Name)
			if previous, equivalent := normalizedValues[normalized]; equivalent {
				return nil, fmt.Errorf("classifier: list %q has equivalent values %q and %q", name, previous, value.Name)
			}
			normalizedValues[normalized] = value.Name
		}
		index[name] = normalizedValues
	}
	return index, nil
}

// validateResponse extracts the response JSON and resolves every list. It
// always returns the list's canonical value (or ""); model spelling is never
// forwarded. Missing lists, non-string values, and unknown values are errors.
func validateResponse(raw string, lists []List, index map[string]listIndex) (map[string]string, error) {
	obj, err := extractObject(raw)
	if err != nil {
		return nil, err
	}
	output := make(map[string]string, len(lists))
	for _, list := range lists {
		name := strings.TrimSpace(list.Name)
		rawValue, present := obj[name]
		if !present {
			return nil, fmt.Errorf("list %q missing from response", name)
		}
		var text string
		if err := json.Unmarshal(rawValue, &text); err != nil {
			return nil, fmt.Errorf("value for list %q is not a string: %v", name, err)
		}
		if strings.TrimSpace(text) == "" {
			output[name] = "" // Explicit empty value means no match.
			continue
		}
		canonical, ok := index[name][normalize(text)]
		if !ok {
			return nil, fmt.Errorf("value %q is not allowed in list %q (allowed: %s)",
				text, name, strings.Join(valueNames(list), ", "))
		}
		output[name] = canonical
	}
	return output, nil
}

func valueNames(l List) []string {
	names := make([]string, 0, len(l.Values))
	for _, value := range l.Values {
		names = append(names, value.Name)
	}
	return names
}

var (
	fenceRe      = regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)```")
	annotationRe = regexp.MustCompile(`\s*\([^)]*\)\s*$`)
)

// extractObject tolerates code fences and surrounding text. Local models emit
// both despite instructions, but the JSON must still decode to an object.
func extractObject(raw string) (map[string]json.RawMessage, error) {
	t := strings.TrimSpace(raw)
	candidates := []string{t}
	for _, match := range fenceRe.FindAllStringSubmatch(t, -1) {
		candidates = append(candidates, match[1])
	}
	if i := strings.Index(t, "{"); i >= 0 {
		if j := strings.LastIndex(t, "}"); j > i {
			candidates = append(candidates, t[i:j+1])
		}
	}
	for _, candidate := range candidates {
		var obj map[string]json.RawMessage
		if err := json.Unmarshal([]byte(strings.TrimSpace(candidate)), &obj); err == nil && obj != nil {
			return obj, nil
		}
	}
	return nil, errors.New("response does not contain a JSON object")
}

// normalize maps model spelling to the canonical value. Case, whitespace, and
// trailing parenthesized annotations do not change the value.
func normalize(s string) string {
	s = annotationRe.ReplaceAllString(strings.TrimSpace(s), "")
	return strings.ToLower(strings.TrimSpace(s))
}
