package validate

import (
	"bufio"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Gabriel-Araujo/network_agent/internal/frr/container"
)

var (
	// Syntax error reported by vtysh to stderr, e.g.:
	//   line 3: % Unknown command[28]:  neighbor 10.0.0.2 bogus-keyword 123
	syntaxLineRE = regexp.MustCompile(`^line (\d+): % (.+)$`)

	// Daemon rejection reported by vtysh --no-fork -f to stderr, e.g.:
	//   line 2: Failure to communicate[13] to bgpd, line:  neighbor 10.0.0.2 description peer
	daemonRejectRE = regexp.MustCompile(`^line (\d+): Failure to communicate\[(\d+)\] to ([a-z0-9_]+), line:\s*(.*)$`)

	// Trailing summary emitted by vtysh when processing fails; not a finding itself.
	processingFailureRE = regexp.MustCompile(`^Configuration file\[[^\]]+\] processing failure: \d+$`)

	// Known stdout warning/notice patterns observed in FRR 10.5.
	missingRouteMapRE = regexp.MustCompile(`^The route-map '([^']+)' does not exist\.?$`)
	deprecatedRE      = regexp.MustCompile(`(?i)^this command is deprecated\b`)
	pleaseUseRE       = regexp.MustCompile(`(?i)^please,?\s+use\b`)
	gracefulRestartRE = regexp.MustCompile(`(?i)^graceful restart configuration changed\b`)

	// Opening markdown code fence (up to 3 leading spaces, 3+ backticks).
	fenceOpenRE = regexp.MustCompile(`^ {0,3}(` + "`{3,}" + `)`)
)

// ParseOutput turns raw stdout/stderr from `vtysh --no-fork -f` or
// `vtysh --dryrun -f` into structured Findings.
//
// file is attached to every finding (may be empty). lineDelta is added to any
// positive 1-indexed line number reported by vtysh (for example,
// markdownStartLine - wrapOffset).
func ParseOutput(stdout, stderr, file string, lineDelta int) []Finding {
	findings := []Finding{}
	var rejectIndices []int
	var rejectCmds []string

	stderrScanner := bufio.NewScanner(strings.NewReader(container.NormalizeLF(stderr)))
	for stderrScanner.Scan() {
		line := strings.TrimSpace(stderrScanner.Text())
		if line == "" || processingFailureRE.MatchString(line) {
			continue
		}
		if m := syntaxLineRE.FindStringSubmatch(line); m != nil {
			n, _ := strconv.Atoi(m[1])
			findings = append(findings, Finding{
				Severity: SeverityError,
				File:     file,
				Line:     adjustLine(n, lineDelta),
				Daemon:   "vtysh",
				Rule:     RuleSyntax,
				Message:  strings.TrimSpace(m[2]),
			})
			continue
		}
		if m := daemonRejectRE.FindStringSubmatch(line); m != nil {
			n, _ := strconv.Atoi(m[1])
			daemon := m[3]
			cmdText := strings.TrimSpace(m[4])
			msg := fmt.Sprintf("daemon %s rejected command", daemon)
			if cmdText != "" {
				msg = fmt.Sprintf("daemon %s rejected command: %s", daemon, cmdText)
			}
			rejectIndices = append(rejectIndices, len(findings))
			rejectCmds = append(rejectCmds, cmdText)
			findings = append(findings, Finding{
				Severity: SeverityError,
				File:     file,
				Line:     adjustLine(n, lineDelta),
				Daemon:   daemon,
				Rule:     RuleDaemonReject,
				Message:  msg,
			})
		}
	}

	var stdoutLines []string
	stdoutScanner := bufio.NewScanner(strings.NewReader(container.NormalizeLF(stdout)))
	for stdoutScanner.Scan() {
		line := strings.TrimSpace(stdoutScanner.Text())
		if line != "" {
			stdoutLines = append(stdoutLines, line)
		}
	}

	var unexplained []string
	for i := 0; i < len(stdoutLines); i++ {
		line := stdoutLines[i]
		switch {
		case missingRouteMapRE.MatchString(line):
			findings = append(findings, Finding{
				Severity: SeverityWarning,
				File:     file,
				Rule:     RuleMissingRouteMap,
				Message:  line,
			})
		case deprecatedRE.MatchString(line):
			msg := line
			if i+1 < len(stdoutLines) && pleaseUseRE.MatchString(stdoutLines[i+1]) {
				msg += " " + stdoutLines[i+1]
				i++
			}
			findings = append(findings, Finding{
				Severity: SeverityWarning,
				File:     file,
				Rule:     RuleDeprecatedCommand,
				Message:  msg,
			})
		case gracefulRestartRE.MatchString(line):
			findings = append(findings, Finding{
				Severity: SeverityInfo,
				File:     file,
				Rule:     RuleDaemonNotice,
				Message:  line,
			})
		default:
			unexplained = append(unexplained, line)
		}
	}

	// Pair human daemon explanations emitted on stdout with daemon-reject
	// findings emitted on stderr (vtysh --no-fork executes sequentially).
	switch {
	case len(unexplained) == 0:
		// nothing left to attach
	case len(rejectIndices) == 1:
		detail := strings.Join(unexplained, "; ")
		idx := rejectIndices[0]
		findings[idx].Message = formatRejectMessage(detail, rejectCmds[0])
	case len(rejectIndices) > 1 && len(unexplained) == len(rejectIndices):
		for i, idx := range rejectIndices {
			findings[idx].Message = formatRejectMessage(unexplained[i], rejectCmds[i])
		}
	default:
		// Ambiguous pairing or no daemon-reject errors: preserve every message
		// as a warning so no daemon diagnostic is ever dropped.
		for _, line := range unexplained {
			findings = append(findings, Finding{
				Severity: SeverityWarning,
				File:     file,
				Rule:     RuleDaemonWarning,
				Message:  line,
			})
		}
	}

	return findings
}

func formatRejectMessage(detail, cmdText string) string {
	if cmdText == "" {
		return detail
	}
	return fmt.Sprintf("%s (%s)", detail, cmdText)
}

func adjustLine(line, delta int) int {
	if line <= 0 {
		return 0
	}
	adjusted := line + delta
	if adjusted < 1 {
		return 1
	}
	return adjusted
}

// ComputePass reports whether a verification passed according to the contract:
// any error fails; warnings also fail when strict is true; info never fails.
func ComputePass(findings []Finding, strict bool) bool {
	for _, f := range findings {
		if f.Severity == SeverityError {
			return false
		}
		if strict && f.Severity == SeverityWarning {
			return false
		}
	}
	return true
}

func hasErrorFinding(findings []Finding) bool {
	for _, f := range findings {
		if f.Severity == SeverityError {
			return true
		}
	}
	return false
}

// ExtractMarkdownConfig extracts the first fenced code block (``` ... ```)
// from a Markdown document.
//
// The returned lineOffset is the 1-indexed line number of the opening fence, so
// line N of the extracted config sits at line N+lineOffset in the original
// Markdown file.
func ExtractMarkdownConfig(markdown string) (config string, lineOffset int, ok bool) {
	lines := strings.Split(container.NormalizeLF(markdown), "\n")
	inFence := false
	fenceTicks := 0
	openLine := 0
	var body []string

	for i, line := range lines {
		lineNum := i + 1
		if !inFence {
			if m := fenceOpenRE.FindStringSubmatch(line); m != nil {
				inFence = true
				fenceTicks = len(m[1])
				openLine = lineNum
			}
			continue
		}

		if isClosingFence(line, fenceTicks) {
			content := strings.Join(body, "\n")
			if len(body) > 0 {
				content += "\n"
			}
			return content, openLine, true
		}
		body = append(body, line)
	}
	return "", 0, false
}

func isClosingFence(line string, minTicks int) bool {
	trimmed := strings.TrimLeft(line, " ")
	leadingSpaces := len(line) - len(trimmed)
	if leadingSpaces > 3 {
		return false
	}
	ticks := 0
	for ticks < len(trimmed) && trimmed[ticks] == '`' {
		ticks++
	}
	if ticks < minTicks {
		return false
	}
	return strings.TrimSpace(trimmed[ticks:]) == ""
}
