package lab

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Gabriel-Araujo/network_agent/internal/frr/container"
)

type fakeExecutor struct {
	fn func(ctx context.Context, name string, args ...string) (container.ExecResult, error)
}

func (f fakeExecutor) Exec(ctx context.Context, name string, args ...string) (container.ExecResult, error) {
	return f.fn(ctx, name, args...)
}

func newFakeRuntime(t *testing.T, fn func(ctx context.Context, name string, args ...string) (container.ExecResult, error)) *container.Runtime {
	t.Helper()
	rt, err := container.Resolve(container.Options{
		Runtime:  "podman",
		Image:    container.DefaultImage,
		Executor: fakeExecutor{fn: fn},
		LookPath: func(string) (string, error) { return "/usr/bin/podman", nil },
	})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	return rt
}

func TestValidateTopology(t *testing.T) {
	valid := Topology{
		Name: "bgp-two-nets",
		Networks: []NetworkSpec{
			{Name: "net-a", Subnet: "10.202.1.0/24"},
			{Name: "net-b", Subnet: "10.202.2.0/24"},
		},
		Nodes: []NodeSpec{
			{Name: "r1", Daemons: []string{"bgpd"}, Interfaces: []InterfaceSpec{{Network: "net-a", IP: "10.202.1.11"}}},
			{Name: "r2", Daemons: []string{"bgpd"}, Interfaces: []InterfaceSpec{
				{Network: "net-a", IP: "10.202.1.12"},
				{Network: "net-b", IP: "10.202.2.12"},
			}},
		},
	}
	if err := validateTopology(valid); err != nil {
		t.Fatalf("validateTopology(valid) unexpected error: %v", err)
	}

	cases := []struct {
		name string
		topo Topology
		want string
	}{
		{
			name: "no networks",
			topo: Topology{Nodes: valid.Nodes},
			want: "at least one network",
		},
		{
			name: "no nodes",
			topo: Topology{Networks: valid.Networks},
			want: "at least one node",
		},
		{
			name: "invalid network name",
			topo: Topology{Networks: []NetworkSpec{{Name: "-bad", Subnet: "10.0.0.0/24"}}, Nodes: valid.Nodes[:1]},
			want: "invalid network name",
		},
		{
			name: "duplicate network",
			topo: Topology{
				Networks: []NetworkSpec{
					{Name: "net-a", Subnet: "10.0.0.0/24"},
					{Name: "net-a", Subnet: "10.0.1.0/24"},
				},
				Nodes: valid.Nodes[:1],
			},
			want: "duplicate network",
		},
		{
			name: "invalid subnet",
			topo: Topology{Networks: []NetworkSpec{{Name: "net-a", Subnet: "not-a-cidr"}}, Nodes: valid.Nodes[:1]},
			want: "invalid subnet",
		},
		{
			name: "duplicate node",
			topo: Topology{
				Networks: valid.Networks[:1],
				Nodes: []NodeSpec{
					{Name: "r1", Interfaces: []InterfaceSpec{{Network: "net-a", IP: "10.202.1.11"}}},
					{Name: "r1", Interfaces: []InterfaceSpec{{Network: "net-a", IP: "10.202.1.12"}}},
				},
			},
			want: "duplicate node",
		},
		{
			name: "node with no interfaces",
			topo: Topology{
				Networks: valid.Networks[:1],
				Nodes:    []NodeSpec{{Name: "r1"}},
			},
			want: "at least one interface",
		},
		{
			name: "unknown network reference",
			topo: Topology{
				Networks: valid.Networks[:1],
				Nodes:    []NodeSpec{{Name: "r1", Interfaces: []InterfaceSpec{{Network: "ghost", IP: "10.202.1.11"}}}},
			},
			want: "unknown network",
		},
		{
			name: "duplicate network on same node",
			topo: Topology{
				Networks: valid.Networks[:1],
				Nodes: []NodeSpec{{
					Name: "r1",
					Interfaces: []InterfaceSpec{
						{Network: "net-a", IP: "10.202.1.11"},
						{Network: "net-a", IP: "10.202.1.12"},
					},
				}},
			},
			want: "more than once",
		},
		{
			name: "IP outside subnet",
			topo: Topology{
				Networks: valid.Networks[:1],
				Nodes:    []NodeSpec{{Name: "r1", Interfaces: []InterfaceSpec{{Network: "net-a", IP: "192.168.1.1"}}}},
			},
			want: "outside network",
		},
		{
			name: "duplicate IP on same network",
			topo: Topology{
				Networks: valid.Networks[:1],
				Nodes: []NodeSpec{
					{Name: "r1", Interfaces: []InterfaceSpec{{Network: "net-a", IP: "10.202.1.11"}}},
					{Name: "r2", Interfaces: []InterfaceSpec{{Network: "net-a", IP: "10.202.1.11"}}},
				},
			},
			want: "assigned to both",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateTopology(tc.topo)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}

func TestStartMultiNetworkAndTwoStageClose(t *testing.T) {
	var calls [][]string
	var closeCtxErrs []error

	rt := newFakeRuntime(t, func(ctx context.Context, _ string, args ...string) (container.ExecResult, error) {
		calls = append(calls, append([]string(nil), args...))
		if len(args) >= 4 && args[0] == "exec" && args[2] == "vtysh" && args[3] == "-c" && args[4] == "show daemons" {
			return container.ExecResult{Stdout: "mgmtd zebra bgpd staticd watchfrr\n"}, nil
		}
		if len(args) >= 1 && (args[0] == "rm" || (args[0] == "network" && len(args) >= 2 && args[1] == "rm")) {
			closeCtxErrs = append(closeCtxErrs, ctx.Err())
		}
		return container.ExecResult{}, nil
	})

	topo := Topology{
		Name: "multi-net",
		Networks: []NetworkSpec{
			{Name: "net-a", Subnet: "10.202.1.0/24"},
			{Name: "net-b", Subnet: "10.202.2.0/24"},
		},
		Nodes: []NodeSpec{
			{
				Name:       "r1",
				Daemons:    []string{"bgpd"},
				Config:     "hostname r1\r\nrouter bgp 65001\r\nexit\r\n",
				Interfaces: []InterfaceSpec{{Network: "net-a", IP: "10.202.1.11"}},
			},
			{
				Name:    "r2",
				Daemons: []string{"bgpd"},
				Config:  "hostname r2\nrouter bgp 65002\nexit\n",
				Interfaces: []InterfaceSpec{
					{Network: "net-a", IP: "10.202.1.12"},
					{Network: "net-b", IP: "10.202.2.12"},
				},
			},
		},
		ReadyTimeout:      2 * time.Second,
		ReadyPollInterval: 5 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())
	l, err := StartWithRuntime(ctx, rt, topo)
	if err != nil {
		t.Fatalf("StartWithRuntime() error = %v", err)
	}

	// Verify temp files were written with LF-only endings.
	rawConf, err := os.ReadFile(filepath.Join(l.tmpDir, "r1", "frr.conf"))
	if err != nil {
		t.Fatalf("read r1 frr.conf: %v", err)
	}
	if strings.Contains(string(rawConf), "\r") {
		t.Errorf("r1 frr.conf contains CR: %q", string(rawConf))
	}

	// Cancel caller context before Close() to verify teardown uses its own context.
	cancel()
	if err := l.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	// Second Close() must be a no-op.
	if err := l.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}

	if _, err := os.Stat(l.tmpDir); !os.IsNotExist(err) {
		t.Errorf("expected temp dir %q to be removed, stat err = %v", l.tmpDir, err)
	}
	for i, ce := range closeCtxErrs {
		if ce != nil {
			t.Errorf("teardown call %d saw cancelled context: %v", i, ce)
		}
	}

	// Inspect command sequence.
	netA, _ := l.NetworkName("net-a")
	netB, _ := l.NetworkName("net-b")
	r1Name, _ := l.ContainerName("r1")
	r2Name, _ := l.ContainerName("r2")

	var foundNetCreate, foundConnect, foundRmContainers, foundRmNetworks bool
	var rmContainersIdx, rmNetworksIdx int
	for idx, c := range calls {
		joined := strings.Join(c, " ")
		if len(c) >= 2 && c[0] == "network" && c[1] == "create" {
			foundNetCreate = true
			if !strings.Contains(joined, "--internal") {
				t.Errorf("network create missing --internal: %s", joined)
			}
			if !strings.Contains(joined, "--label frr-harness.managed=true") || !strings.Contains(joined, "--label frr-harness.run="+l.RunID()) {
				t.Errorf("network create missing harness labels: %s", joined)
			}
		}
		if len(c) >= 1 && c[0] == "create" {
			for _, want := range []string{
				"--cap-add=NET_ADMIN",
				"--cap-add=NET_RAW",
				"--cap-add=SYS_ADMIN",
				"--ulimit nofile=4096:4096",
				"--label frr-harness.managed=true",
				":/etc/frr/daemons:ro",
				":/etc/frr/vtysh.conf:ro",
				":/etc/frr/frr.conf:ro",
			} {
				if !strings.Contains(joined, want) {
					t.Errorf("container create missing %q in %s", want, joined)
				}
			}
			if strings.Contains(joined, "--privileged") {
				t.Errorf("container create must never pass --privileged: %s", joined)
			}
			if strings.Contains(joined, `\`) {
				t.Errorf("container create must normalize Windows backslashes: %s", joined)
			}
		}
		if slices.Equal(c, []string{"network", "connect", "--ip", "10.202.2.12", netB, r2Name}) {
			foundConnect = true
		}
		if slices.Equal(c, []string{"rm", "-f", "-t", "0", r1Name, r2Name}) {
			foundRmContainers = true
			rmContainersIdx = idx
		}
		if slices.Equal(c, []string{"network", "rm", netA, netB}) {
			foundRmNetworks = true
			rmNetworksIdx = idx
		}
	}

	if !foundNetCreate {
		t.Error("did not see network create call")
	}
	if !foundConnect {
		t.Errorf("did not see secondary network connect call for r2 on %s; calls=%#v", netB, calls)
	}
	if !foundRmContainers || !foundRmNetworks {
		t.Fatalf("missing teardown calls: rmContainers=%v rmNetworks=%v", foundRmContainers, foundRmNetworks)
	}
	if rmContainersIdx >= rmNetworksIdx {
		t.Errorf("containers must be removed before networks: rmContainersIdx=%d rmNetworksIdx=%d", rmContainersIdx, rmNetworksIdx)
	}
}

func TestStartPartialFailureCleansUpAutomatically(t *testing.T) {
	var calls [][]string
	createCount := 0

	rt := newFakeRuntime(t, func(_ context.Context, _ string, args ...string) (container.ExecResult, error) {
		calls = append(calls, append([]string(nil), args...))
		if len(args) > 0 && args[0] == "create" {
			createCount++
			if createCount == 2 {
				return container.ExecResult{Stderr: "simulated create failure", ExitCode: 125}, errors.New("exit 125")
			}
		}
		return container.ExecResult{}, nil
	})

	topo := Topology{
		Networks: []NetworkSpec{{Name: "net-a", Subnet: "10.202.1.0/24"}},
		Nodes: []NodeSpec{
			{Name: "r1", Daemons: []string{"bgpd"}, Interfaces: []InterfaceSpec{{Network: "net-a", IP: "10.202.1.11"}}},
			{Name: "r2", Daemons: []string{"bgpd"}, Interfaces: []InterfaceSpec{{Network: "net-a", IP: "10.202.1.12"}}},
		},
	}

	_, err := StartWithRuntime(context.Background(), rt, topo)
	if err == nil {
		t.Fatal("expected StartWithRuntime to fail on second container create")
	}

	var sawRmContainer, sawRmNetwork bool
	for _, c := range calls {
		if len(c) >= 5 && c[0] == "rm" && c[1] == "-f" && c[2] == "-t" && c[3] == "0" && strings.HasSuffix(c[4], "-r1") {
			sawRmContainer = true
		}
		if len(c) >= 3 && c[0] == "network" && c[1] == "rm" && strings.HasSuffix(c[2], "-net-a") {
			sawRmNetwork = true
		}
	}
	if !sawRmContainer || !sawRmNetwork {
		t.Fatalf("expected automatic teardown of r1 and net-a on partial failure; calls=%#v", calls)
	}
}

func TestSweepOrphansWithRuntime(t *testing.T) {
	t.Run("removes managed containers before managed networks", func(t *testing.T) {
		var calls [][]string
		rt := newFakeRuntime(t, func(_ context.Context, _ string, args ...string) (container.ExecResult, error) {
			calls = append(calls, append([]string(nil), args...))
			if len(args) >= 2 && args[0] == "ps" && args[1] == "-aq" {
				return container.ExecResult{Stdout: "c1\nc2\n"}, nil
			}
			if len(args) >= 3 && args[0] == "network" && args[1] == "ls" {
				return container.ExecResult{Stdout: "n1\n"}, nil
			}
			return container.ExecResult{}, nil
		})

		if err := SweepOrphansWithRuntime(context.Background(), rt); err != nil {
			t.Fatalf("SweepOrphansWithRuntime() error = %v", err)
		}

		want := [][]string{
			{"ps", "-aq", "--filter", "label=frr-harness.managed=true"},
			{"rm", "-f", "-t", "0", "c1", "c2"},
			{"network", "ls", "-q", "--filter", "label=frr-harness.managed=true"},
			{"network", "rm", "n1"},
		}
		if len(calls) != len(want) {
			t.Fatalf("calls len = %d (%#v), want %d (%#v)", len(calls), calls, len(want), want)
		}
		for i := range want {
			if !slices.Equal(calls[i], want[i]) {
				t.Errorf("call[%d] = %#v, want %#v", i, calls[i], want[i])
			}
		}
	})

	t.Run("does not invoke rm when no orphans exist", func(t *testing.T) {
		var calls [][]string
		rt := newFakeRuntime(t, func(_ context.Context, _ string, args ...string) (container.ExecResult, error) {
			calls = append(calls, append([]string(nil), args...))
			return container.ExecResult{Stdout: "\n"}, nil
		})

		if err := SweepOrphansWithRuntime(context.Background(), rt); err != nil {
			t.Fatalf("SweepOrphansWithRuntime() error = %v", err)
		}
		if len(calls) != 2 {
			t.Fatalf("expected only ps and network ls calls, got %#v", calls)
		}
	})
}

func TestWaitFor(t *testing.T) {
	t.Run("succeeds on third attempt", func(t *testing.T) {
		attempts := 0
		err := WaitFor(context.Background(), time.Second, 5*time.Millisecond, func(context.Context) (bool, error) {
			attempts++
			return attempts >= 3, nil
		})
		if err != nil {
			t.Fatalf("WaitFor() unexpected error: %v", err)
		}
		if attempts != 3 {
			t.Errorf("attempts = %d, want 3", attempts)
		}
	})

	t.Run("aborts immediately on condition error", func(t *testing.T) {
		boom := errors.New("fatal parse error")
		attempts := 0
		err := WaitFor(context.Background(), time.Second, 5*time.Millisecond, func(context.Context) (bool, error) {
			attempts++
			return false, boom
		})
		if !errors.Is(err, boom) {
			t.Fatalf("WaitFor() err = %v, want %v", err, boom)
		}
		if attempts != 1 {
			t.Errorf("attempts = %d, want 1", attempts)
		}
	})

	t.Run("returns deadline exceeded on timeout", func(t *testing.T) {
		err := WaitFor(context.Background(), 30*time.Millisecond, 5*time.Millisecond, func(context.Context) (bool, error) {
			return false, nil
		})
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("WaitFor() err = %v, want DeadlineExceeded", err)
		}
	})
}

func TestBGPJSONParsersFRR105Payloads(t *testing.T) {
	t.Run("BGPSummary parses multiPathRelax string and peerState Policy vs OK", func(t *testing.T) {
		const rawOK = `{
			"routerId":"10.202.0.11",
			"as":65001,
			"vrfId":0,
			"vrfName":"default",
			"tableVersion":1,
			"ribCount":1,
			"ribMemory":128,
			"peerCount":1,
			"peers":{
				"10.202.0.12":{
					"hostname":"r2",
					"remoteAs":65002,
					"localAs":65001,
					"version":4,
					"msgRcvd":4,
					"msgSent":5,
					"pfxRcd":0,
					"pfxSnt":1,
					"state":"Established",
					"peerState":"OK",
					"connectionsEstablished":1,
					"idType":"ipv4"
				},
				"10.202.0.13":{
					"remoteAs":65003,
					"localAs":65001,
					"pfxRcd":0,
					"pfxSnt":0,
					"state":"Established",
					"peerState":"Policy"
				}
			},
			"totalPeers":2,
			"bestPath":{"multiPathRelax":"false"}
		}`

		var sum BGPSummary
		if err := json.Unmarshal([]byte(rawOK), &sum); err != nil {
			t.Fatalf("Unmarshal BGPSummary error = %v", err)
		}
		if sum.BestPath.MultiPathRelax != "false" {
			t.Errorf("MultiPathRelax = %q, want \"false\"", sum.BestPath.MultiPathRelax)
		}
		pOK, ok := sum.Peer("10.202.0.12")
		if !ok || !pOK.IsEstablished() || !pOK.IsPolicyOK() || pOK.IsPolicyBlocked() || pOK.PfxSnt != 1 {
			t.Errorf("unexpected OK peer: %#v", pOK)
		}
		pPol, ok := sum.Peer("10.202.0.13")
		if !ok || !pPol.IsEstablished() || pPol.IsPolicyOK() || !pPol.IsPolicyBlocked() {
			t.Errorf("unexpected Policy peer: %#v", pPol)
		}
	})

	t.Run("BGPRoutes requires both Valid and Bestpath (import-check trap)", func(t *testing.T) {
		const raw = `{
			"vrfId": 0,
			"vrfName": "default",
			"routerId": "10.202.0.11",
			"localAS": 65001,
			"routes": {
				"192.168.10.0/24": [{
					"valid": true,
					"bestpath": true,
					"pathFrom": "external",
					"prefix": "192.168.10.0",
					"prefixLen": 24,
					"network": "192.168.10.0/24",
					"version": 1,
					"peerId": "10.202.0.11",
					"path": "65001",
					"origin": "IGP",
					"nexthops": [{"ip": "10.202.0.11", "hostname": "r1", "afi": "ipv4", "used": true}]
				}],
				"192.168.99.0/24": [{
					"prefix": "192.168.99.0",
					"prefixLen": 24,
					"network": "192.168.99.0/24",
					"version": 0,
					"origin": "IGP"
				}]
			},
			"totalRoutes": 2,
			"totalPaths": 2
		}`

		var routes BGPRoutes
		if err := json.Unmarshal([]byte(raw), &routes); err != nil {
			t.Fatalf("Unmarshal BGPRoutes error = %v", err)
		}
		if !routes.HasValidBestPath("192.168.10.0/24") {
			t.Errorf("expected 192.168.10.0/24 to have valid bestpath")
		}
		if routes.HasValidBestPath("192.168.99.0/24") {
			t.Errorf("192.168.99.0/24 without valid/bestpath must NOT pass HasValidBestPath")
		}
		if routes.HasValidBestPath("192.168.77.0/24") {
			t.Errorf("absent prefix must NOT pass HasValidBestPath")
		}
	})

	t.Run("AdvertisedRoutes decodes object map and empty map", func(t *testing.T) {
		const rawFull = `{
			"bgpTableVersion":1,
			"bgpLocalRouterId":"10.202.0.11",
			"localAS":65001,
			"advertisedRoutes":{
				"192.168.10.0/24":{
					"addrPrefix":"192.168.10.0",
					"prefixLen":24,
					"network":"192.168.10.0/24",
					"nextHop":"0.0.0.0",
					"weight":32768,
					"origin":"IGP",
					"valid":true,
					"best":true
				}
			},
			"totalPrefixCounter":1,
			"filteredPrefixCounter":0
		}`
		var adv AdvertisedRoutes
		if err := json.Unmarshal([]byte(rawFull), &adv); err != nil {
			t.Fatalf("Unmarshal AdvertisedRoutes error = %v", err)
		}
		if !adv.HasPrefix("192.168.10.0/24") || adv.TotalPrefixCounter != 1 {
			t.Errorf("unexpected AdvertisedRoutes: %#v", adv)
		}

		const rawEmpty = `{"bgpTableVersion":0,"advertisedRoutes":{},"totalPrefixCounter":0,"filteredPrefixCounter":0}`
		var empty AdvertisedRoutes
		if err := json.Unmarshal([]byte(rawEmpty), &empty); err != nil {
			t.Fatalf("Unmarshal empty AdvertisedRoutes error = %v", err)
		}
		if empty.HasPrefix("192.168.10.0/24") || empty.TotalPrefixCounter != 0 {
			t.Errorf("unexpected empty AdvertisedRoutes: %#v", empty)
		}
	})

	t.Run("IPRoutes checks installed BGP and static blackhole entries", func(t *testing.T) {
		const raw = `{
			"192.168.10.0/24":[{
				"prefix":"192.168.10.0/24",
				"prefixLen":24,
				"protocol":"bgp",
				"selected":true,
				"destSelected":true,
				"distance":20,
				"installed":true,
				"nexthops":[{"ip":"10.202.0.11","afi":"ipv4","interfaceName":"eth0","active":true,"fib":true}]
			}],
			"192.168.20.0/24":[{
				"prefix":"192.168.20.0/24",
				"prefixLen":24,
				"protocol":"static",
				"selected":true,
				"installed":true,
				"nexthops":[{"unreachable":true,"blackhole":true,"active":true,"fib":true}]
			}]
		}`
		var rib IPRoutes
		if err := json.Unmarshal([]byte(raw), &rib); err != nil {
			t.Fatalf("Unmarshal IPRoutes error = %v", err)
		}
		if !rib.HasInstalled("192.168.10.0/24", "bgp") {
			t.Error("expected 192.168.10.0/24 installed as bgp")
		}
		if !rib.HasInstalledVia("192.168.10.0/24", "bgp", "10.202.0.11") {
			t.Error("expected 192.168.10.0/24 installed via 10.202.0.11")
		}
		if rib.HasInstalledVia("192.168.10.0/24", "bgp", "10.202.0.99") {
			t.Error("unexpected match for wrong nexthop")
		}
		if rib.HasInstalled("192.168.20.0/24", "bgp") {
			t.Error("static route must not match protocol bgp")
		}
		if !rib.HasInstalled("192.168.20.0/24", "static") {
			t.Error("expected static blackhole route to match protocol static")
		}
	})
}
