package validate

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/Gabriel-Araujo/network_agent/internal/frr/container"
)

type fakeExec struct {
	fn func(ctx context.Context, name string, args ...string) (container.ExecResult, error)
}

func (f fakeExec) Exec(ctx context.Context, name string, args ...string) (container.ExecResult, error) {
	return f.fn(ctx, name, args...)
}

func fakeRuntimeOpts(fn func(ctx context.Context, name string, args ...string) (container.ExecResult, error)) container.Options {
	return container.Options{
		Runtime:  "podman",
		Image:    container.DefaultImage,
		LookPath: func(file string) (string, error) { return "/usr/bin/" + file, nil },
		Getenv:   func(string) string { return "" },
		Executor: fakeExec{fn: fn},
	}
}

// The wrapper must keep the snippet's line numbering recoverable: snippet
// line N must sit at wrapped line N+offset, so callers can subtract the
// returned offset from vtysh-reported lines.
func TestWrapSnippetLineMapping(t *testing.T) {
	// leading blank line on purpose: wrapping must not trim the snippet
	snippet := "\nneighbor 10.0.0.1 remote-as 65001\n  network 10.0.0.0/24\n"
	wrapped, offset := WrapSnippet("bgp", snippet)
	if offset != 1 {
		t.Fatalf("offset = %d, want 1", offset)
	}
	lines := strings.Split(wrapped, "\n")
	for i, want := range strings.Split(snippet, "\n") {
		got := lines[i+offset]
		if got != " "+want {
			t.Errorf("snippet line %d at wrapped[%d] = %q, want %q", i+1, i+offset, got, " "+want)
		}
	}
}

func TestWrapSnippetNoWrap(t *testing.T) {
	cases := []struct {
		name, protocol, snippet string
	}{
		{"already has context", "bgp", "router bgp 65000\n  neighbor 10.0.0.1 remote-as 65001\n"},
		{"interface context", "bgp", "interface eth0\n  ip address 10.0.0.1/24\n"},
		{"unknown protocol", "static", "ip route 10.0.0.0/24 10.0.0.1\n"},
		{"empty", "bgp", "  \n"},
	}
	for _, c := range cases {
		wrapped, offset := WrapSnippet(c.protocol, c.snippet)
		if offset != 0 {
			t.Errorf("%s: offset = %d, want 0", c.name, offset)
		}
		if wrapped != c.snippet {
			t.Errorf("%s: snippet was modified: %q", c.name, wrapped)
		}
	}
}

func TestParseOutput(t *testing.T) {
	t.Run("syntax and daemon semantic error combined with line delta", func(t *testing.T) {
		stdout := "% Invalid command. Not an internal neighbor\n"
		stderr := strings.Join([]string{
			"line 3: % Unknown command[28]:  neighbor 10.0.0.2 bogus-keyword 123",
			"line 4: Failure to communicate[13] to bgpd, line:  neighbor 10.0.0.2 route-reflector-client",
			"Configuration file[/etc/frr/frr.conf] processing failure: 13",
		}, "\n")

		got := ParseOutput(stdout, stderr, "bgp.md", 7)
		if len(got) != 2 {
			t.Fatalf("len(findings) = %d, want 2: %#v", len(got), got)
		}

		if got[0].Severity != SeverityError || got[0].Line != 10 || got[0].Daemon != "vtysh" || got[0].Rule != RuleSyntax || got[0].File != "bgp.md" {
			t.Errorf("finding[0] = %#v", got[0])
		}
		if got[1].Severity != SeverityError || got[1].Line != 11 || got[1].Daemon != "bgpd" || got[1].Rule != RuleDaemonReject {
			t.Errorf("finding[1] = %#v", got[1])
		}
		if !strings.Contains(got[1].Message, "Not an internal neighbor") || !strings.Contains(got[1].Message, "route-reflector-client") {
			t.Errorf("finding[1].Message = %q, want daemon explanation + command", got[1].Message)
		}
	})

	t.Run("multiple stdout lines attached to single daemon reject", func(t *testing.T) {
		stdout := "BGP instance name and AS number mismatch\nBGP instance is already running; AS is 65001\n"
		stderr := "line 5: Failure to communicate[13] to bgpd, line: router bgp 65002\n"

		got := ParseOutput(stdout, stderr, "dup.conf", 0)
		if len(got) != 1 {
			t.Fatalf("len(findings) = %d, want 1: %#v", len(got), got)
		}
		if got[0].Line != 5 || got[0].Daemon != "bgpd" || got[0].Rule != RuleDaemonReject {
			t.Errorf("finding = %#v", got[0])
		}
		if !strings.Contains(got[0].Message, "AS number mismatch") || !strings.Contains(got[0].Message, "AS is 65001") {
			t.Errorf("message = %q", got[0].Message)
		}
	})

	t.Run("warnings and notices classified distinctly", func(t *testing.T) {
		stdout := strings.Join([]string{
			"The route-map 'RM-OUT' does not exist.",
			"This command is deprecated, because it is not VRF-aware.",
			`Please, use "no ip ospf passive" on an interface instead.`,
			"Graceful restart configuration changed, reset all peers to take effect",
			"Some unexpected daemon note",
		}, "\n")

		got := ParseOutput(stdout, "", "sample.conf", 0)
		if len(got) != 4 {
			t.Fatalf("len(findings) = %d, want 4: %#v", len(got), got)
		}
		if got[0].Severity != SeverityWarning || got[0].Rule != RuleMissingRouteMap {
			t.Errorf("finding[0] = %#v", got[0])
		}
		if got[1].Severity != SeverityWarning || got[1].Rule != RuleDeprecatedCommand || !strings.Contains(got[1].Message, "no ip ospf passive") {
			t.Errorf("finding[1] = %#v", got[1])
		}
		if got[2].Severity != SeverityInfo || got[2].Rule != RuleDaemonNotice {
			t.Errorf("finding[2] = %#v", got[2])
		}
		if got[3].Severity != SeverityWarning || got[3].Rule != RuleDaemonWarning {
			t.Errorf("finding[3] = %#v", got[3])
		}
	})

	t.Run("negative line delta clamps to 1", func(t *testing.T) {
		stderr := "line 1: % Unknown command[28]: router bgp\n"
		got := ParseOutput("", stderr, "snippet", -1)
		if len(got) != 1 || got[0].Line != 1 {
			t.Fatalf("got = %#v, want line 1", got)
		}
	})
}

func TestComputePass(t *testing.T) {
	infoOnly := []Finding{{Severity: SeverityInfo, Rule: RuleDaemonNotice, Message: "gr"}}
	warnAndInfo := []Finding{
		{Severity: SeverityInfo, Rule: RuleDaemonNotice, Message: "gr"},
		{Severity: SeverityWarning, Rule: RuleMissingRouteMap, Message: "missing"},
	}
	withErr := []Finding{{Severity: SeverityError, Rule: RuleDaemonReject, Message: "boom"}}

	if !ComputePass(nil, false) || !ComputePass(nil, true) {
		t.Error("empty findings should pass in both modes")
	}
	if !ComputePass(infoOnly, false) || !ComputePass(infoOnly, true) {
		t.Error("info findings must never fail, even under strict")
	}
	if !ComputePass(warnAndInfo, false) {
		t.Error("warnings should pass when strict=false")
	}
	if ComputePass(warnAndInfo, true) {
		t.Error("warnings must fail when strict=true")
	}
	if ComputePass(withErr, false) || ComputePass(withErr, true) {
		t.Error("errors must always fail")
	}
}

func TestExtractMarkdownConfig(t *testing.T) {
	md := "# Title\r\n\r\nIntro text.\r\n\r\n```frr\r\nrouter bgp 65001\r\n neighbor 10.0.0.2 description peer\r\n```\r\n\r\n```bash\r\necho ignored\r\n```\r\n"
	cfg, offset, ok := ExtractMarkdownConfig(md)
	if !ok {
		t.Fatal("expected fenced block to be found")
	}
	if offset != 5 {
		t.Fatalf("offset = %d, want 5", offset)
	}
	wantCfg := "router bgp 65001\n neighbor 10.0.0.2 description peer\n"
	if cfg != wantCfg {
		t.Fatalf("cfg = %q, want %q", cfg, wantCfg)
	}

	if _, _, ok := ExtractMarkdownConfig("# No code block here\n"); ok {
		t.Fatal("expected ok=false when markdown has no fenced code block")
	}
}

func TestBuildRunArgs(t *testing.T) {
	tf := tempFiles{
		vtyshConf: `C:\tmp\vtysh.conf`,
		checkConf: `C:\tmp\check.conf`,
		daemons:   `C:\tmp\daemons`,
		emptyConf: `C:\tmp\frr.conf`,
		runScript: `C:\tmp\run.sh`,
	}

	daemonArgs := strings.Join(buildRunArgs(ModeDaemon, container.DefaultImage, "run1", tf), " ")
	for _, want := range []string{
		"run --rm --network=none",
		"--cap-add=NET_ADMIN",
		"--cap-add=NET_RAW",
		"--cap-add=SYS_ADMIN",
		"--ulimit nofile=4096:4096",
		"--label frr-harness.managed=true",
		"--label frr-harness.run=run1",
		"C:/tmp/daemons:/etc/frr/daemons:ro",
		"C:/tmp/vtysh.conf:/etc/frr/vtysh.conf:ro",
		"C:/tmp/frr.conf:/etc/frr/frr.conf:ro",
		"C:/tmp/check.conf:/tmp/check.conf:ro",
		"C:/tmp/run.sh:/tmp/run.sh:ro",
		container.DefaultImage + " /bin/sh /tmp/run.sh",
	} {
		if !strings.Contains(daemonArgs, want) {
			t.Errorf("daemon args missing %q:\n%s", want, daemonArgs)
		}
	}
	if strings.Contains(daemonArgs, "--privileged") {
		t.Errorf("daemon args must never contain --privileged: %s", daemonArgs)
	}

	grammarArgs := strings.Join(buildRunArgs(ModeGrammar, container.DefaultImage, "run2", tf), " ")
	for _, want := range []string{
		"run --rm --network=none",
		"--label frr-harness.managed=true",
		"--label frr-harness.run=run2",
		"--entrypoint /usr/bin/vtysh",
		"C:/tmp/vtysh.conf:/etc/frr/vtysh.conf:ro",
		"C:/tmp/check.conf:/tmp/check.conf:ro",
		container.DefaultImage + " --dryrun -f /tmp/check.conf",
	} {
		if !strings.Contains(grammarArgs, want) {
			t.Errorf("grammar args missing %q:\n%s", want, grammarArgs)
		}
	}
	if strings.Contains(grammarArgs, "--cap-add") {
		t.Errorf("grammar args should not require capabilities: %s", grammarArgs)
	}
}

func TestCheckWithFakeExecutor(t *testing.T) {
	t.Run("daemon semantic error returns Pass=false and nil error", func(t *testing.T) {
		opts := Options{
			Mode: ModeDaemon,
			File: "test.conf",
			Runtime: fakeRuntimeOpts(func(_ context.Context, _ string, _ ...string) (container.ExecResult, error) {
				return container.ExecResult{
					Stdout:   "% Specify remote-as or peer-group commands first\n",
					Stderr:   "line 2: Failure to communicate[13] to bgpd, line:  neighbor 10.0.0.2 description peer\nConfiguration file[/etc/frr/frr.conf] processing failure: 13\n",
					ExitCode: 13,
				}, errors.New("exit status 13")
			}),
		}
		rep, err := Check(context.Background(), "router bgp 65001\n neighbor 10.0.0.2 description peer\n", opts)
		if err != nil {
			t.Fatalf("Check() unexpected error = %v", err)
		}
		if rep.Pass {
			t.Fatal("expected Pass=false")
		}
		if rep.Mode != ModeDaemon || len(rep.Findings) != 1 || rep.Findings[0].Daemon != "bgpd" || rep.Findings[0].Line != 2 {
			t.Fatalf("unexpected report: %#v", rep)
		}

		raw, err := json.Marshal(rep)
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}
		if strings.Contains(string(raw), `"valid"`) {
			t.Errorf("report JSON must not contain legacy valid field: %s", string(raw))
		}
	})

	t.Run("non-zero exit without parseable error finding returns Go error", func(t *testing.T) {
		opts := Options{
			Mode: ModeDaemon,
			Runtime: fakeRuntimeOpts(func(_ context.Context, _ string, args ...string) (container.ExecResult, error) {
				if len(args) > 0 && args[0] == "run" {
					return container.ExecResult{
						Stderr:   "frr-harness: daemons did not become ready within timeout\n",
						ExitCode: 124,
					}, errors.New("exit status 124")
				}
				return container.ExecResult{}, nil
			}),
		}
		rep, err := Check(context.Background(), "hostname r1\n", opts)
		if err == nil {
			t.Fatalf("expected operational error, got nil (report=%#v)", rep)
		}
		if !strings.Contains(err.Error(), "exit 124") {
			t.Errorf("err = %v, want exit 124 details", err)
		}
	})

	t.Run("operational failure surfaces ErrImageMissing when image absent", func(t *testing.T) {
		opts := Options{
			Mode: ModeGrammar,
			Runtime: fakeRuntimeOpts(func(_ context.Context, _ string, args ...string) (container.ExecResult, error) {
				if len(args) > 0 && args[0] == "run" {
					return container.ExecResult{Stderr: "pull access denied", ExitCode: 125}, errors.New("exit 125")
				}
				if len(args) >= 2 && args[0] == "image" && args[1] == "inspect" {
					return container.ExecResult{Stderr: "image not known", ExitCode: 125}, errors.New("exit 125")
				}
				return container.ExecResult{}, nil
			}),
		}
		_, err := Check(context.Background(), "hostname r1\n", opts)
		if !errors.Is(err, container.ErrImageMissing) {
			t.Fatalf("err = %v, want ErrImageMissing", err)
		}
	})
}

func TestCheckBatchOrdering(t *testing.T) {
	var mu sync.Mutex
	seen := 0

	opts := Options{
		Mode: ModeDaemon,
		Runtime: fakeRuntimeOpts(func(_ context.Context, _ string, args ...string) (container.ExecResult, error) {
			if len(args) > 0 && (args[0] == "info" || args[0] == "image") {
				return container.ExecResult{}, nil
			}
			mu.Lock()
			seen++
			mu.Unlock()
			return container.ExecResult{
				Stdout:   "The route-map 'RM-X' does not exist.\n",
				ExitCode: 0,
			}, nil
		}),
	}

	inputs := []Input{
		{File: "a.conf", Config: "hostname a\n"},
		{File: "b.conf", Config: "hostname b\n"},
		{File: "c.conf", Config: "hostname c\n"},
	}

	rep, err := CheckBatch(context.Background(), inputs, opts, 2)
	if err != nil {
		t.Fatalf("CheckBatch() error = %v", err)
	}
	if !rep.Pass {
		t.Fatalf("expected Pass=true with warnings when strict=false")
	}
	if seen != 3 {
		t.Errorf("container runs = %d, want 3", seen)
	}
	if len(rep.Findings) != 3 {
		t.Fatalf("len(Findings) = %d, want 3", len(rep.Findings))
	}
	wantOrder := []string{"a.conf", "b.conf", "c.conf"}
	for i, wantFile := range wantOrder {
		if rep.Files[i] != wantFile {
			t.Errorf("Files[%d] = %q, want %q", i, rep.Files[i], wantFile)
		}
		if rep.Findings[i].File != wantFile {
			t.Errorf("Findings[%d].File = %q, want %q", i, rep.Findings[i].File, wantFile)
		}
	}
}
