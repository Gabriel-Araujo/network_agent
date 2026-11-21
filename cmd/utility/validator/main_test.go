package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Gabriel-Araujo/network_agent/internal/frr/validate"
)

func TestCLIMarkdownAutoAndJSONAndExitCodes(t *testing.T) {
	dir := t.TempDir()
	mdPath := filepath.Join(dir, "example.md")
	mdContent := "# Example\n\n```frr\nrouter bgp 65001\n neighbor 10.0.0.2 description peer\n```\n"
	if err := os.WriteFile(mdPath, []byte(mdContent), 0o644); err != nil {
		t.Fatalf("write md: %v", err)
	}

	var gotInputs []validate.Input
	var gotOpts validate.Options
	var gotJobs int

	fakeCheck := func(_ context.Context, inputs []validate.Input, opts validate.Options, jobs int) (*validate.Report, error) {
		gotInputs = inputs
		gotOpts = opts
		gotJobs = jobs
		return &validate.Report{
			Pass:  false,
			Mode:  opts.Mode,
			Files: []string{inputs[0].File},
			Findings: []validate.Finding{{
				Severity: validate.SeverityError,
				File:     inputs[0].File,
				Line:     2 + inputs[0].LineDelta,
				Daemon:   "bgpd",
				Rule:     validate.RuleDaemonReject,
				Message:  "% Specify remote-as or peer-group commands first",
			}},
		}, nil
	}

	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"--json", "--jobs", "2", "--strict", mdPath}, strings.NewReader(""), &stdout, &stderr, fakeCheck)
	if code != exitFindings {
		t.Fatalf("exit code = %d, want %d (stderr=%q)", code, exitFindings, stderr.String())
	}
	if len(gotInputs) != 1 || gotInputs[0].LineDelta != 3 {
		t.Fatalf("inputs = %#v, want 1 input with LineDelta=3", gotInputs)
	}
	if !strings.HasPrefix(gotInputs[0].Config, "router bgp 65001\n") {
		t.Fatalf("extracted config = %q", gotInputs[0].Config)
	}
	if gotOpts.Mode != validate.ModeDaemon || !gotOpts.Strict || gotJobs != 2 {
		t.Fatalf("opts=%#v jobs=%d", gotOpts, gotJobs)
	}

	var rep validate.Report
	if err := json.Unmarshal(stdout.Bytes(), &rep); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, stdout.String())
	}
	if rep.Pass || len(rep.Findings) != 1 || rep.Findings[0].Line != 5 {
		t.Fatalf("decoded report = %#v, want finding on line 5", rep)
	}
}

func TestCLIStdinPassAndProtocolWrap(t *testing.T) {
	var gotInputs []validate.Input
	fakeCheck := func(_ context.Context, inputs []validate.Input, opts validate.Options, _ int) (*validate.Report, error) {
		gotInputs = inputs
		return &validate.Report{
			Pass:     true,
			Mode:     opts.Mode,
			Findings: []validate.Finding{},
		}, nil
	}

	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"--mode", "grammar", "--protocol", "bgp"}, strings.NewReader("neighbor 10.0.0.2 remote-as 65002\n"), &stdout, &stderr, fakeCheck)
	if code != exitPass {
		t.Fatalf("exit code = %d, want 0 (stderr=%q)", code, stderr.String())
	}
	if len(gotInputs) != 1 || gotInputs[0].LineDelta != -1 {
		t.Fatalf("inputs = %#v, want LineDelta=-1 from WrapSnippet", gotInputs)
	}
	if !strings.Contains(stdout.String(), "PASS (grammar, stdin:") {
		t.Errorf("unexpected human output: %q", stdout.String())
	}
}

func TestCLIOperationalAndUsageExitCode2(t *testing.T) {
	noop := func(_ context.Context, _ []validate.Input, _ validate.Options, _ int) (*validate.Report, error) {
		return nil, errors.New("podman machine stopped")
	}

	cases := []struct {
		name string
		args []string
		in   string
	}{
		{"invalid mode", []string{"--mode", "bogus"}, ""},
		{"invalid jobs", []string{"--jobs", "0"}, ""},
		{"missing file", []string{filepath.Join(t.TempDir(), "does-not-exist.conf")}, ""},
		{"markdown without code block", []string{"--markdown"}, "# no fence here\n"},
		{"runtime operational error", []string{}, "hostname r1\n"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run(context.Background(), c.args, strings.NewReader(c.in), &stdout, &stderr, noop)
			if code != exitOperational {
				t.Errorf("exit code = %d, want %d (stdout=%q stderr=%q)", code, exitOperational, stdout.String(), stderr.String())
			}
		})
	}
}
