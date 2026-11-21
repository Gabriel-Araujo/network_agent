// Command validator runs Layer 1 FRR verification (Carga em daemon by default,
// or Checagem de gramática with --mode grammar) over stdin or one/more files,
// automatically extracting the first fenced code block from .md files.
//
// Exit codes:
//   - 0: approved (no blocking findings)
//   - 1: rejected by findings
//   - 2: usage error or container/runtime operational failure
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Gabriel-Araujo/network_agent/internal/frr/container"
	"github.com/Gabriel-Araujo/network_agent/internal/frr/validate"
)

const (
	exitPass        = 0
	exitFindings    = 1
	exitOperational = 2
)

type batchChecker func(ctx context.Context, inputs []validate.Input, opts validate.Options, jobs int) (*validate.Report, error)

func main() {
	code := run(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr, validate.CheckBatch)
	os.Exit(code)
}

func run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer, checkFn batchChecker) int {
	fs := flag.NewFlagSet("validator", flag.ContinueOnError)
	fs.SetOutput(stderr)

	modeFlag := fs.String("mode", string(validate.ModeDaemon), "verification mode: daemon (Carga em daemon) or grammar (Checagem de gramática)")
	strictFlag := fs.Bool("strict", false, "treat warning findings as failures (info findings never fail)")
	jsonFlag := fs.Bool("json", false, "emit structured JSON report")
	jobsFlag := fs.Int("jobs", validate.DefaultJobs, "max concurrent containers for multi-file batches")
	protocolFlag := fs.String("protocol", "", "protocol hint (bgp, ospf, isis...) to wrap a bare snippet")
	runtimeFlag := fs.String("runtime", "", "container runtime override (e.g. podman, docker)")
	imageFlag := fs.String("image", "", "FRR container image override")
	timeoutFlag := fs.Duration("timeout", validate.DefaultTimeout, "per-file verification timeout")
	markdownFlag := fs.Bool("markdown", false, "extract first fenced code block (auto-enabled for .md files)")
	fileFlag := fs.String("file", "", "config file to check (optional; positional paths also supported; defaults to stdin)")

	if err := fs.Parse(args); err != nil {
		return exitOperational
	}

	mode, err := validate.ParseMode(*modeFlag)
	if err != nil {
		fmt.Fprintln(stderr, "validator:", err)
		return exitOperational
	}
	if *jobsFlag < 1 {
		fmt.Fprintf(stderr, "validator: --jobs must be >= 1 (got %d)\n", *jobsFlag)
		return exitOperational
	}
	if *timeoutFlag < 0 {
		fmt.Fprintf(stderr, "validator: --timeout must be >= 0 (got %v)\n", *timeoutFlag)
		return exitOperational
	}

	var paths []string
	if *fileFlag != "" {
		paths = append(paths, *fileFlag)
	}
	paths = append(paths, fs.Args()...)

	inputs, err := loadInputs(paths, stdin, *markdownFlag, *protocolFlag)
	if err != nil {
		fmt.Fprintln(stderr, "validator:", err)
		return exitOperational
	}

	opts := validate.Options{
		Mode:   mode,
		Strict: *strictFlag,
		Runtime: container.Options{
			Runtime: *runtimeFlag,
			Image:   *imageFlag,
		},
		Timeout: *timeoutFlag,
	}

	report, err := checkFn(ctx, inputs, opts, *jobsFlag)
	if err != nil {
		fmt.Fprintln(stderr, "validator:", err)
		return exitOperational
	}

	if *jsonFlag {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintln(stderr, "validator: encode json:", err)
			return exitOperational
		}
	} else {
		writeHumanReport(stdout, report, len(paths) == 0)
	}

	if !report.Pass {
		return exitFindings
	}
	return exitPass
}

func loadInputs(paths []string, stdin io.Reader, forceMarkdown bool, protocol string) ([]validate.Input, error) {
	if len(paths) == 0 {
		data, err := io.ReadAll(stdin)
		if err != nil {
			return nil, fmt.Errorf("read stdin: %w", err)
		}
		in, err := prepareInput("", string(data), forceMarkdown, protocol)
		if err != nil {
			return nil, err
		}
		return []validate.Input{in}, nil
	}

	inputs := make([]validate.Input, 0, len(paths))
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", p, err)
		}
		useMarkdown := forceMarkdown || strings.EqualFold(filepath.Ext(p), ".md")
		in, err := prepareInput(p, string(data), useMarkdown, protocol)
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, in)
	}
	return inputs, nil
}

func prepareInput(file, raw string, useMarkdown bool, protocol string) (validate.Input, error) {
	config := raw
	lineDelta := 0

	if useMarkdown {
		extracted, mdOffset, ok := validate.ExtractMarkdownConfig(raw)
		if !ok {
			label := file
			if label == "" {
				label = "stdin"
			}
			return validate.Input{}, fmt.Errorf("no fenced code block found in %s", label)
		}
		config = extracted
		lineDelta += mdOffset
	}

	if protocol != "" {
		wrapped, wrapOffset := validate.WrapSnippet(protocol, config)
		config = wrapped
		lineDelta -= wrapOffset
	}

	return validate.Input{
		File:      file,
		Config:    config,
		LineDelta: lineDelta,
	}, nil
}

func writeHumanReport(w io.Writer, report *validate.Report, fromStdin bool) {
	status := "PASS"
	if !report.Pass {
		status = "FAIL"
	}

	var errs, warns, infos int
	for _, f := range report.Findings {
		switch f.Severity {
		case validate.SeverityError:
			errs++
		case validate.SeverityWarning:
			warns++
		case validate.SeverityInfo:
			infos++
		}
	}

	scope := "stdin"
	if !fromStdin {
		n := len(report.Files)
		if n == 1 {
			scope = report.Files[0]
		} else {
			scope = fmt.Sprintf("%d files", n)
		}
	}

	fmt.Fprintf(w, "%s (%s, %s: %d error, %d warning, %d info)\n",
		status, report.Mode, scope, errs, warns, infos)

	for _, f := range report.Findings {
		loc := formatLocation(f.File, f.Line)
		meta := formatMeta(f.Daemon, f.Rule)
		switch {
		case loc != "" && meta != "":
			fmt.Fprintf(w, "  [%s] %s (%s): %s\n", f.Severity, loc, meta, f.Message)
		case loc != "":
			fmt.Fprintf(w, "  [%s] %s: %s\n", f.Severity, loc, f.Message)
		case meta != "":
			fmt.Fprintf(w, "  [%s] (%s): %s\n", f.Severity, meta, f.Message)
		default:
			fmt.Fprintf(w, "  [%s] %s\n", f.Severity, f.Message)
		}
	}
}

func formatLocation(file string, line int) string {
	switch {
	case file != "" && line > 0:
		return fmt.Sprintf("%s:%d", file, line)
	case file != "":
		return file
	case line > 0:
		return fmt.Sprintf("line %d", line)
	default:
		return ""
	}
}

func formatMeta(daemon, rule string) string {
	switch {
	case daemon != "" && rule != "":
		return daemon + "/" + rule
	case daemon != "":
		return daemon
	case rule != "":
		return rule
	default:
		return ""
	}
}
