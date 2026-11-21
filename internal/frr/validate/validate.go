// Package validate implements Layer 1 of the FRR verification harness:
// single-file verification in throwaway FRR containers.
//
// It supports two verification levels defined in CONTEXT.md ("Verificação FRR"):
//   - ModeDaemon ("daemon", default — Carga em daemon): boots active FRR
//     daemons in a throwaway --network=none container and loads frr.conf via
//     `vtysh --no-fork -f`, catching grammar errors, daemon semantic rejections,
//     and daemon warnings.
//   - ModeGrammar ("grammar" — Checagem de gramática): runs `vtysh --dryrun -f`
//     with no daemons running, checking only compiled-in command grammar.
//
// Neither level proves live network or control-plane behavior (that belongs to
// Layer 2 / Prova de topologia).
package validate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Gabriel-Araujo/network_agent/internal/frr/container"
)

// Mode selects the Layer 1 verification level.
type Mode string

const (
	// ModeDaemon runs Carga em daemon against active FRR daemons.
	ModeDaemon Mode = "daemon"
	// ModeGrammar runs Checagem de gramática via vtysh --dryrun.
	ModeGrammar Mode = "grammar"
)

// Severity classifies a diagnostic Finding (Achado).
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// Canonical finding rules emitted by Layer 1.
const (
	RuleSyntax            = "syntax"
	RuleDaemonReject      = "daemon-reject"
	RuleMissingRouteMap   = "missing-route-map"
	RuleDeprecatedCommand = "deprecated-command"
	RuleDaemonNotice      = "daemon-notice"
	RuleDaemonWarning     = "daemon-warning"
)

// DefaultJobs is the default worker pool concurrency for CheckBatch.
const DefaultJobs = 4

// DefaultTimeout is applied per verification call when Options.Timeout is zero
// and the caller's context has no deadline.
const DefaultTimeout = 60 * time.Second

// Finding is one diagnostic item (Achado) emitted by an FRR verification.
type Finding struct {
	Severity Severity `json:"severity"`
	File     string   `json:"file,omitempty"`
	Line     int      `json:"line,omitempty"`
	Daemon   string   `json:"daemon,omitempty"`
	Rule     string   `json:"rule,omitempty"`
	Message  string   `json:"message"`
}

// Report is the structured verification outcome, aligned with Stage
// 05-validate's report.json contract.
type Report struct {
	Pass     bool      `json:"pass"`
	Mode     Mode      `json:"mode"`
	Files    []string  `json:"files,omitempty"`
	Findings []Finding `json:"findings"`
	ExitCode int       `json:"exit_code,omitempty"`
	Stdout   string    `json:"stdout,omitempty"`
	Stderr   string    `json:"stderr,omitempty"`
}

// Options configures a Layer 1 verification run.
type Options struct {
	// Mode selects ModeDaemon (default when empty) or ModeGrammar.
	Mode Mode
	// Strict causes warning findings to fail Pass in addition to errors.
	// Info findings never fail Pass.
	Strict bool
	// File attaches a logical filename to Findings and Report.Files.
	File string
	// LineDelta is added to 1-indexed line numbers reported by vtysh
	// (for example, markdownOffset - wrapOffset).
	LineDelta int
	// Daemons optionally overrides the optional FRR daemons started in
	// ModeDaemon. When nil, all 21 daemons (container.AllDaemons + core) start.
	Daemons []string
	// Runtime configures container runtime/image resolution.
	Runtime container.Options
	// Timeout caps each single-file container invocation (0 uses DefaultTimeout
	// when ctx has no deadline).
	Timeout time.Duration
}

// Input represents one configuration file in a batch run.
type Input struct {
	File      string
	Config    string
	LineDelta int
}

const daemonRunnerScript = `#!/bin/sh
set -u
/usr/lib/frr/frrinit.sh start >/tmp/boot.log 2>&1
i=0
while [ "$i" -lt 300 ]; do
	if /usr/lib/frr/frrinit.sh status >/dev/null 2>&1 \
		&& ! pgrep -f "vtysh" >/dev/null 2>&1 \
		&& vtysh -c "configure terminal" -c "end" >/dev/null 2>&1; then
		exec vtysh --no-fork -f /tmp/check.conf
	fi
	sleep 0.05
	i=$((i + 1))
done
echo "frr-harness: daemons did not become ready within timeout" >&2
cat /tmp/boot.log >&2 2>/dev/null || true
/usr/lib/frr/frrinit.sh status >&2 || true
exit 124
`

// ParseMode validates and normalizes a verification mode string.
func ParseMode(s string) (Mode, error) {
	switch Mode(strings.ToLower(strings.TrimSpace(s))) {
	case "", ModeDaemon:
		return ModeDaemon, nil
	case ModeGrammar:
		return ModeGrammar, nil
	default:
		return "", fmt.Errorf("invalid verification mode %q: want %q or %q", s, ModeDaemon, ModeGrammar)
	}
}

// Check runs Layer 1 verification on a single FRR configuration string.
//
// It returns a Go error only on operational failures (invalid options, missing
// runtime/image, container startup failure, timeout, or non-zero exit with no
// parseable error finding). Configuration defects return a non-nil Report with
// Pass == false and err == nil.
func Check(ctx context.Context, configText string, opts Options) (*Report, error) {
	mode, err := normalizeMode(opts.Mode)
	if err != nil {
		return nil, err
	}
	opts.Mode = mode

	rt, err := container.Resolve(opts.Runtime)
	if err != nil {
		return nil, err
	}
	return CheckWithRuntime(ctx, rt, configText, opts)
}

// CheckWithRuntime runs Layer 1 verification using an already-resolved container
// Runtime. Safe for concurrent calls across goroutines.
func CheckWithRuntime(ctx context.Context, rt *container.Runtime, configText string, opts Options) (*Report, error) {
	if rt == nil {
		return nil, errors.New("container runtime is nil")
	}
	mode, err := normalizeMode(opts.Mode)
	if err != nil {
		return nil, err
	}
	opts.Mode = mode

	tmpDir, err := os.MkdirTemp("", "frr-validate-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	files, err := writeTempFiles(tmpDir, configText, opts)
	if err != nil {
		return nil, err
	}

	runID := container.NewRunID()
	args := buildRunArgs(mode, rt.Image, runID, files)

	runCtx, cancel := withEffectiveTimeout(ctx, opts.Timeout)
	defer cancel()

	res, runErr := rt.Exec(runCtx, args...)

	findings := ParseOutput(res.Stdout, res.Stderr, opts.File, opts.LineDelta)
	report := &Report{
		Mode:     mode,
		Findings: findings,
		ExitCode: res.ExitCode,
		Stdout:   res.Stdout,
		Stderr:   res.Stderr,
	}
	if opts.File != "" {
		report.Files = []string{opts.File}
	}

	if runCtx.Err() != nil {
		return report, fmt.Errorf("frr verification (%s) aborted: %w", mode, runCtx.Err())
	}

	if runErr != nil && !hasErrorFinding(findings) {
		if checkErr := rt.Check(ctx); checkErr != nil {
			return report, checkErr
		}
		detail := strings.TrimSpace(res.Stderr)
		if detail == "" {
			detail = strings.TrimSpace(res.Stdout)
		}
		if detail != "" {
			return report, fmt.Errorf("frr verification (%s) failed operationally (exit %d): %s", mode, res.ExitCode, detail)
		}
		return report, fmt.Errorf("frr verification (%s) failed operationally (exit %d): %w", mode, res.ExitCode, runErr)
	}

	report.Pass = ComputePass(findings, opts.Strict)
	return report, nil
}

// CheckBatch verifies multiple inputs concurrently using a worker pool of
// throwaway containers (jobs defaults to DefaultJobs when <= 0). Findings and
// Files are merged in the exact order of inputs.
func CheckBatch(ctx context.Context, inputs []Input, opts Options, jobs int) (*Report, error) {
	mode, err := normalizeMode(opts.Mode)
	if err != nil {
		return nil, err
	}
	opts.Mode = mode

	if jobs <= 0 {
		jobs = DefaultJobs
	}

	rt, err := container.Resolve(opts.Runtime)
	if err != nil {
		return nil, err
	}
	if err := rt.Check(ctx); err != nil {
		return nil, err
	}

	if len(inputs) == 0 {
		return &Report{
			Pass:     true,
			Mode:     mode,
			Findings: []Finding{},
		}, nil
	}
	if len(inputs) == 1 {
		singleOpts := opts
		singleOpts.File = inputs[0].File
		singleOpts.LineDelta = inputs[0].LineDelta
		return CheckWithRuntime(ctx, rt, inputs[0].Config, singleOpts)
	}

	if jobs > len(inputs) {
		jobs = len(inputs)
	}

	batchCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	type slot struct {
		report *Report
		err    error
	}
	results := make([]slot, len(inputs))
	work := make(chan int, len(inputs))
	for i := range inputs {
		work <- i
	}
	close(work)

	var wg sync.WaitGroup
	wg.Add(jobs)
	for range jobs {
		go func() {
			defer wg.Done()
			for idx := range work {
				if batchCtx.Err() != nil {
					return
				}
				itemOpts := opts
				itemOpts.File = inputs[idx].File
				itemOpts.LineDelta = inputs[idx].LineDelta
				rep, err := CheckWithRuntime(batchCtx, rt, inputs[idx].Config, itemOpts)
				results[idx] = slot{report: rep, err: err}
				if err != nil {
					cancel()
					return
				}
			}
		}()
	}
	wg.Wait()

	merged := &Report{
		Mode:     mode,
		Files:    make([]string, 0, len(inputs)),
		Findings: []Finding{},
	}
	for i, s := range results {
		if s.err != nil {
			return nil, fmt.Errorf("%s: %w", inputs[i].File, s.err)
		}
		if inputs[i].File != "" {
			merged.Files = append(merged.Files, inputs[i].File)
		}
		if s.report != nil {
			merged.Findings = append(merged.Findings, s.report.Findings...)
			if merged.ExitCode == 0 && s.report.ExitCode != 0 {
				merged.ExitCode = s.report.ExitCode
			}
		}
	}
	merged.Pass = ComputePass(merged.Findings, opts.Strict)
	return merged, nil
}

type tempFiles struct {
	vtyshConf string
	checkConf string
	daemons   string
	emptyConf string
	runScript string
}

func writeTempFiles(dir, configText string, opts Options) (tempFiles, error) {
	tf := tempFiles{
		vtyshConf: filepath.Join(dir, "vtysh.conf"),
		checkConf: filepath.Join(dir, "check.conf"),
	}
	if err := container.WriteLF(tf.vtyshConf, container.DefaultVtyshConf, 0o644); err != nil {
		return tempFiles{}, err
	}
	normalized := container.NormalizeLF(configText)
	if normalized != "" && !strings.HasSuffix(normalized, "\n") {
		normalized += "\n"
	}
	if err := container.WriteLF(tf.checkConf, normalized, 0o644); err != nil {
		return tempFiles{}, err
	}

	if opts.Mode == ModeDaemon {
		tf.daemons = filepath.Join(dir, "daemons")
		tf.emptyConf = filepath.Join(dir, "frr.conf")
		tf.runScript = filepath.Join(dir, "run.sh")

		daemons := opts.Daemons
		if daemons == nil {
			daemons = container.AllDaemons
		}
		if err := container.WriteLF(tf.daemons, container.RenderDaemons(daemons), 0o644); err != nil {
			return tempFiles{}, err
		}
		if err := container.WriteLF(tf.emptyConf, "", 0o644); err != nil {
			return tempFiles{}, err
		}
		if err := container.WriteLF(tf.runScript, daemonRunnerScript, 0o755); err != nil {
			return tempFiles{}, err
		}
	}
	return tf, nil
}

func buildRunArgs(mode Mode, image, runID string, tf tempFiles) []string {
	if mode == ModeGrammar {
		args := []string{"run", "--rm", "--network=none"}
		args = append(args, container.Labels(runID)...)
		args = append(args,
			"--entrypoint", "/usr/bin/vtysh",
			"-v", container.BindMount(tf.vtyshConf, "/etc/frr/vtysh.conf", true),
			"-v", container.BindMount(tf.checkConf, "/tmp/check.conf", true),
			image,
			"--dryrun", "-f", "/tmp/check.conf",
		)
		return args
	}

	args := []string{"run", "--rm", "--network=none"}
	args = append(args, container.BaseRunArgs(runID)...)
	args = append(args,
		"-v", container.BindMount(tf.daemons, "/etc/frr/daemons", true),
		"-v", container.BindMount(tf.vtyshConf, "/etc/frr/vtysh.conf", true),
		"-v", container.BindMount(tf.emptyConf, "/etc/frr/frr.conf", true),
		"-v", container.BindMount(tf.checkConf, "/tmp/check.conf", true),
		"-v", container.BindMount(tf.runScript, "/tmp/run.sh", true),
		image,
		"/bin/sh", "/tmp/run.sh",
	)
	return args
}

func normalizeMode(m Mode) (Mode, error) {
	if m == "" {
		return ModeDaemon, nil
	}
	return ParseMode(string(m))
}

func withEffectiveTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout > 0 {
		return context.WithTimeout(ctx, timeout)
	}
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, DefaultTimeout)
}

// contextWrappers maps a protocol to the minimal FRR config context its
// second-level commands (neighbor, network, area...) need to be nested
// inside to verify standalone. Best-effort: covers single-level nesting only.
var contextWrappers = map[string]string{
	"bgp":   "router bgp 65000",
	"ospf":  "router ospf",
	"ospf6": "router ospf6",
	"isis":  "router isis WORD",
	"rip":   "router rip",
	"ripng": "router ripng",
	"pim":   "router pim",
}

// WrapSnippet wraps a bare config snippet in the context it needs to be checked
// on its own, if it doesn't already carry that context (a line starting with
// "router " or "interface "). Snippets for protocols with no wrapper in
// contextWrappers (e.g. static routes, which are top-level) are returned
// unchanged. The returned int is how many lines the wrapper prepends (0 when
// unchanged) — subtract it from line numbers reported against the wrapped text
// to recover the line in the original snippet.
func WrapSnippet(protocol, snippet string) (string, int) {
	trimmed := strings.TrimSpace(snippet)
	if trimmed == "" {
		return snippet, 0
	}
	firstLine := strings.SplitN(trimmed, "\n", 2)[0]
	if strings.HasPrefix(firstLine, "router ") || strings.HasPrefix(firstLine, "interface ") {
		return snippet, 0
	}
	wrapper, ok := contextWrappers[protocol]
	if !ok || wrapper == "" {
		return snippet, 0
	}

	var b strings.Builder
	b.WriteString(wrapper)
	b.WriteString("\n")
	for _, line := range strings.Split(snippet, "\n") {
		b.WriteString(" ")
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("!\n")
	return b.String(), 1
}
