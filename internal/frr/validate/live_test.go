package validate

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/Gabriel-Araujo/network_agent/internal/frr/container"
)

func requireLiveRuntime(t *testing.T) *container.Runtime {
	t.Helper()
	if os.Getenv(container.EnvLive) != "1" {
		t.Skipf("live FRR test: set %s=1 to run against local container runtime", container.EnvLive)
	}
	rt, err := container.Resolve(container.Options{})
	if err != nil {
		t.Fatalf("%s=1 set, but container runtime resolution failed: %v", container.EnvLive, err)
	}
	if err := rt.Check(context.Background()); err != nil {
		t.Fatalf("%s=1 set, but runtime/image check failed: %v", container.EnvLive, err)
	}
	return rt
}

// TestLiveGrammarVsDaemonSemanticGap proves the core distinction in CONTEXT.md:
// Checagem de gramática (grammar) accepts `neighbor ... description` without
// `remote-as`, whereas Carga em daemon (daemon) rejects it with exit 13 and a
// structured bgpd Finding on the exact line.
func TestLiveGrammarVsDaemonSemanticGap(t *testing.T) {
	rt := requireLiveRuntime(t)
	cfg := strings.Join([]string{
		"router bgp 65001",
		" neighbor 10.0.0.2 description peer-without-remote-as",
		"exit",
	}, "\n") + "\n"

	grammarRep, err := CheckWithRuntime(context.Background(), rt, cfg, Options{
		Mode: ModeGrammar,
		File: "no-remote-as.conf",
	})
	if err != nil {
		t.Fatalf("grammar check operational error: %v", err)
	}
	if !grammarRep.Pass {
		t.Fatalf("expected grammar mode to pass on semantic-only defect, got findings=%#v", grammarRep.Findings)
	}

	daemonRep, err := CheckWithRuntime(context.Background(), rt, cfg, Options{
		Mode: ModeDaemon,
		File: "no-remote-as.conf",
	})
	if err != nil {
		t.Fatalf("daemon check operational error: %v", err)
	}
	if daemonRep.Pass {
		t.Fatalf("expected daemon mode to fail on missing remote-as, got pass (stdout=%q stderr=%q)", daemonRep.Stdout, daemonRep.Stderr)
	}
	if len(daemonRep.Findings) == 0 {
		t.Fatal("expected at least one Finding")
	}
	f := daemonRep.Findings[0]
	if f.Severity != SeverityError || f.Daemon != "bgpd" || f.Rule != RuleDaemonReject || f.Line != 2 {
		t.Errorf("unexpected finding: %#v", f)
	}
	if !strings.Contains(f.Message, "remote-as") {
		t.Errorf("expected daemon explanation mentioning remote-as, got %q", f.Message)
	}
}

func TestLiveDaemonWarningsNoticesAndRPKI(t *testing.T) {
	rt := requireLiveRuntime(t)

	t.Run("missing route-map warns and fails only under strict", func(t *testing.T) {
		cfg := strings.Join([]string{
			"router bgp 65001",
			" neighbor 10.0.0.2 remote-as 65002",
			" address-family ipv4 unicast",
			"  neighbor 10.0.0.2 route-map RM-MISSING out",
			" exit-address-family",
			"exit",
		}, "\n") + "\n"

		rep, err := CheckWithRuntime(context.Background(), rt, cfg, Options{Mode: ModeDaemon})
		if err != nil {
			t.Fatalf("CheckWithRuntime() error = %v", err)
		}
		if !rep.Pass {
			t.Fatalf("expected Pass=true without strict, findings=%#v", rep.Findings)
		}
		if len(rep.Findings) != 1 || rep.Findings[0].Severity != SeverityWarning || rep.Findings[0].Rule != RuleMissingRouteMap {
			t.Fatalf("unexpected findings: %#v", rep.Findings)
		}

		strictRep, err := CheckWithRuntime(context.Background(), rt, cfg, Options{Mode: ModeDaemon, Strict: true})
		if err != nil {
			t.Fatalf("CheckWithRuntime(strict) error = %v", err)
		}
		if strictRep.Pass {
			t.Fatalf("expected Pass=false under strict for missing route-map")
		}
	})

	t.Run("rpki module loaded and graceful-restart notice stays info under strict", func(t *testing.T) {
		cfg := strings.Join([]string{
			"rpki",
			" rpki polling_period 300",
			" rpki cache tcp 192.0.2.10 3323 preference 1",
			"exit",
			"router bgp 65001",
			" bgp graceful-restart",
			" neighbor 10.0.0.2 remote-as 65002",
			"exit",
		}, "\n") + "\n"

		rep, err := CheckWithRuntime(context.Background(), rt, cfg, Options{Mode: ModeDaemon, Strict: true})
		if err != nil {
			t.Fatalf("CheckWithRuntime() error = %v", err)
		}
		if !rep.Pass {
			t.Fatalf("expected Pass=true even under strict, findings=%#v", rep.Findings)
		}
		foundNotice := false
		for _, f := range rep.Findings {
			if f.Severity == SeverityInfo && f.Rule == RuleDaemonNotice {
				foundNotice = true
			}
		}
		if !foundNotice {
			t.Errorf("expected graceful-restart info notice, got findings=%#v", rep.Findings)
		}
	})

	t.Run("deprecated command produces warning with coalesced guidance", func(t *testing.T) {
		cfg := "router ospf\n no passive-interface eth1\nexit\n"
		rep, err := CheckWithRuntime(context.Background(), rt, cfg, Options{Mode: ModeDaemon})
		if err != nil {
			t.Fatalf("CheckWithRuntime() error = %v", err)
		}
		if !rep.Pass {
			t.Fatalf("expected Pass=true without strict, findings=%#v", rep.Findings)
		}
		if len(rep.Findings) != 1 || rep.Findings[0].Severity != SeverityWarning || rep.Findings[0].Rule != RuleDeprecatedCommand {
			t.Fatalf("unexpected findings: %#v", rep.Findings)
		}
		if !strings.Contains(rep.Findings[0].Message, "no ip ospf passive") {
			t.Errorf("expected coalesced replacement guidance, got %q", rep.Findings[0].Message)
		}
	})

	t.Run("syntax error caught with exact line", func(t *testing.T) {
		cfg := "router bgp 65001\n neighbor 10.0.0.2 bogus-keyword 123\nexit\n"
		rep, err := CheckWithRuntime(context.Background(), rt, cfg, Options{Mode: ModeDaemon})
		if err != nil {
			t.Fatalf("CheckWithRuntime() error = %v", err)
		}
		if rep.Pass {
			t.Fatal("expected syntax error to fail")
		}
		if len(rep.Findings) == 0 || rep.Findings[0].Rule != RuleSyntax || rep.Findings[0].Line != 2 {
			t.Fatalf("unexpected findings: %#v", rep.Findings)
		}
	})
}

func TestLiveCheckBatch(t *testing.T) {
	requireLiveRuntime(t)

	inputs := []Input{
		{File: "ok.conf", Config: "hostname ok\nrouter bgp 65001\n neighbor 10.0.0.2 remote-as 65002\nexit\n"},
		{File: "bad.conf", Config: "router bgp 65001\n neighbor 10.0.0.2 description missing-as\nexit\n"},
	}

	rep, err := CheckBatch(context.Background(), inputs, Options{Mode: ModeDaemon}, 2)
	if err != nil {
		t.Fatalf("CheckBatch() error = %v", err)
	}
	if rep.Pass {
		t.Fatalf("expected batch to fail because bad.conf has semantic error")
	}
	if len(rep.Files) != 2 || rep.Files[0] != "ok.conf" || rep.Files[1] != "bad.conf" {
		t.Fatalf("unexpected Files: %#v", rep.Files)
	}
	if len(rep.Findings) != 1 || rep.Findings[0].File != "bad.conf" || rep.Findings[0].Rule != RuleDaemonReject {
		t.Fatalf("unexpected Findings: %#v", rep.Findings)
	}
}
