// Package validate checks whether a candidate FRR configuration is
// syntactically valid before it's shown to the user — the last line of
// defense against an LLM confidently generating a config block that looks
// right but would fail (or misconfigure) a real router.
//
// It shells out to `vtysh -C` (check-only / dry-run mode). This does NOT
// require any FRR daemon to be running: vtysh validates against its own
// compiled-in command grammar, so a bare `vtysh` binary is enough — no
// zebra/bgpd/ospfd process, no container, no socket to manage.
package validate

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// ConfigError is one syntax error reported by vtysh, tied to the line in
// the submitted config that failed.
type ConfigError struct {
	Line    int
	Message string
}

// Result is the outcome of validating a candidate FRR config.
type Result struct {
	Valid  bool
	Errors []ConfigError
}

// vtysh -C prints errors to stderr as "line N: % <message>", one per line,
// for every failing line in the file (not just the first).
var errorLineRE = regexp.MustCompile(`^line (\d+): % (.+)$`)

// ValidateConfig writes configText to a temp file, runs `vtysh -C -f
// <file>`, and parses stderr for errors. Safe for concurrent use — each
// call gets its own temp file, and the config is passed as a file argument
// (never interpolated into a shell string), so there's no injection risk
// even with untrusted/LLM-generated input.
func ValidateConfig(ctx context.Context, configText string) (*Result, error) {
	tmp, err := os.CreateTemp("", "frr-validate-*.conf")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.WriteString(configText); err != nil {
		tmp.Close()
		return nil, fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return nil, fmt.Errorf("close temp file: %w", err)
	}

	cmd := exec.CommandContext(ctx, "vtysh", "-C", "-f", tmp.Name())
	var stderr strings.Builder
	cmd.Stderr = &stderr
	runErr := cmd.Run()

	result := &Result{Valid: runErr == nil}
	scanner := bufio.NewScanner(strings.NewReader(stderr.String()))
	for scanner.Scan() {
		if m := errorLineRE.FindStringSubmatch(scanner.Text()); m != nil {
			lineNum, _ := strconv.Atoi(m[1])
			result.Errors = append(result.Errors, ConfigError{Line: lineNum, Message: m[2]})
		}
	}

	// runErr non-nil for a reason we couldn't parse (vtysh missing, temp
	// file unreadable, etc.) -- surface it instead of silently reporting
	// "invalid, no errors".
	if runErr != nil && len(result.Errors) == 0 {
		return result, fmt.Errorf("vtysh failed without a parseable syntax error: %w (stderr: %s)", runErr, stderr.String())
	}
	return result, nil
}

// contextWrappers maps a protocol to the minimal FRR config context its
// second-level commands (neighbor, network, area...) need to be nested
// inside to validate standalone -- this matters because chunks stored in
// frr_docs (see chunker.Chunk.Content) never include their parent context;
// a command_reference chunk for "neighbor ... remote-as ..." is just that
// one line. Best-effort: covers single-level nesting only. Commands that
// need two levels (e.g. inside an address-family block) must already
// include their own wrapper in the snippet.
var contextWrappers = map[string]string{
	"bgp":   "router bgp 65000",
	"ospf":  "router ospf",
	"ospf6": "router ospf6",
	"isis":  "router isis WORD",
	"rip":   "router rip",
	"ripng": "router ripng",
	"pim":   "router pim",
}

// WrapSnippet wraps a bare config snippet in the context it needs to
// validate on its own, if it doesn't already carry that context (a line
// starting with "router " or "interface "). Snippets for protocols with no
// wrapper in contextWrappers (e.g. static routes, which are top-level) are
// returned unchanged.
func WrapSnippet(protocol, snippet string) string {
	trimmed := strings.TrimSpace(snippet)
	if trimmed == "" {
		return snippet
	}
	firstLine := strings.SplitN(trimmed, "\n", 2)[0]
	if strings.HasPrefix(firstLine, "router ") || strings.HasPrefix(firstLine, "interface ") {
		return snippet
	}
	wrapper, ok := contextWrappers[protocol]
	if !ok || wrapper == "" {
		return snippet
	}

	var b strings.Builder
	b.WriteString(wrapper)
	b.WriteString("\n")
	for _, line := range strings.Split(trimmed, "\n") {
		b.WriteString(" ")
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("!\n")
	return b.String()
}
