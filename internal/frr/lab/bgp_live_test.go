package lab

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Gabriel-Araujo/network_agent/internal/frr/container"
)

const liveConvergeTimeout = 20 * time.Second

func requireLiveRuntime(t *testing.T) *container.Runtime {
	t.Helper()
	if os.Getenv(container.EnvLive) != "1" {
		t.Skipf("live FRR topology proof: set %s=1 to run against local container runtime", container.EnvLive)
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

func startLab(t *testing.T, rt *container.Runtime, topo Topology) *Lab {
	t.Helper()
	l, err := StartWithRuntime(context.Background(), rt, topo)
	if err != nil {
		t.Fatalf("StartWithRuntime(%q) error = %v", topo.Name, err)
	}
	t.Cleanup(func() {
		if err := l.Close(); err != nil {
			t.Errorf("Lab(%q).Close() error = %v", topo.Name, err)
		}
	})
	return l
}

// TestLiveBGPBaseSession proves basic two-node eBGP session convergence with
// default FRR timers (no timer mutation) and BGP route installation into the
// peer's zebra RIB.
func TestLiveBGPBaseSession(t *testing.T) {
	rt := requireLiveRuntime(t)
	ctx := context.Background()

	l := startLab(t, rt, Topology{
		Name:     "bgp-base",
		Networks: []NetworkSpec{{Name: "link", Subnet: "10.203.1.0/24"}},
		Nodes: []NodeSpec{
			{
				Name:       "r1",
				Daemons:    []string{"bgpd"},
				Interfaces: []InterfaceSpec{{Network: "link", IP: "10.203.1.11"}},
				Config: `frr defaults traditional
hostname r1
!
ip prefix-list ALLOW-ALL seq 10 permit 0.0.0.0/0 le 32
!
ip route 192.168.10.0/24 blackhole
!
router bgp 65001
 bgp router-id 10.203.1.11
 neighbor 10.203.1.12 remote-as 65002
 !
 address-family ipv4 unicast
  network 192.168.10.0/24
  neighbor 10.203.1.12 prefix-list ALLOW-ALL in
  neighbor 10.203.1.12 prefix-list ALLOW-ALL out
 exit-address-family
exit
`,
			},
			{
				Name:       "r2",
				Daemons:    []string{"bgpd"},
				Interfaces: []InterfaceSpec{{Network: "link", IP: "10.203.1.12"}},
				Config: `frr defaults traditional
hostname r2
!
ip prefix-list ALLOW-ALL seq 10 permit 0.0.0.0/0 le 32
!
router bgp 65002
 bgp router-id 10.203.1.12
 neighbor 10.203.1.11 remote-as 65001
 !
 address-family ipv4 unicast
  neighbor 10.203.1.11 prefix-list ALLOW-ALL in
  neighbor 10.203.1.11 prefix-list ALLOW-ALL out
 exit-address-family
exit
`,
			},
		},
	})

	p1, err := l.WaitForBGPEstablished(ctx, "r1", "10.203.1.12", liveConvergeTimeout)
	if err != nil {
		t.Fatal(err)
	}
	if !p1.IsPolicyOK() {
		t.Fatalf("expected r1 peerState=OK, got %q", p1.PeerState)
	}

	if err := l.WaitForIPRoute(ctx, "r2", "192.168.10.0/24", "bgp", liveConvergeTimeout); err != nil {
		t.Fatal(err)
	}

	rib, err := l.IPRoutes(ctx, "r2")
	if err != nil {
		t.Fatalf("r2 IPRoutes() error = %v", err)
	}
	if !rib.HasInstalledVia("192.168.10.0/24", "bgp", "10.203.1.11") {
		t.Fatalf("expected 192.168.10.0/24 installed via 10.203.1.11 in r2 RIB, got %#v", rib["192.168.10.0/24"])
	}
}

// TestLiveBGPRFC8212Policy proves FRR 10.5.0's RFC 8212 enforcement under
// `frr defaults traditional` (`bgp ebgp-requires-policy`):
//  1. No policy: session reaches Established with peerState "Policy", 0 prefixes exchanged.
//  2. Only prefix-list (in & out, no route-map): satisfies RFC 8212 (peerState "OK"),
//     filters outbound and inbound prefixes as declared.
//  3. Asymmetric policy (out only): advertises outbound, but keeps peerState "Policy"
//     and discards inbound updates.
func TestLiveBGPRFC8212Policy(t *testing.T) {
	rt := requireLiveRuntime(t)
	ctx := context.Background()

	t.Run("no_policy_blocks_exchange_with_peerState_Policy", func(t *testing.T) {
		l := startLab(t, rt, Topology{
			Name:     "rfc8212-no-policy",
			Networks: []NetworkSpec{{Name: "link", Subnet: "10.203.2.0/24"}},
			Nodes: []NodeSpec{
				{
					Name:       "r1",
					Daemons:    []string{"bgpd"},
					Interfaces: []InterfaceSpec{{Network: "link", IP: "10.203.2.11"}},
					Config: `frr defaults traditional
hostname r1
!
ip route 192.168.10.0/24 blackhole
!
router bgp 65001
 bgp router-id 10.203.2.11
 neighbor 10.203.2.12 remote-as 65002
 !
 address-family ipv4 unicast
  network 192.168.10.0/24
 exit-address-family
exit
`,
				},
				{
					Name:       "r2",
					Daemons:    []string{"bgpd"},
					Interfaces: []InterfaceSpec{{Network: "link", IP: "10.203.2.12"}},
					Config: `frr defaults traditional
hostname r2
!
ip route 192.168.20.0/24 blackhole
!
router bgp 65002
 bgp router-id 10.203.2.12
 neighbor 10.203.2.11 remote-as 65001
 !
 address-family ipv4 unicast
  network 192.168.20.0/24
 exit-address-family
exit
`,
				},
			},
		})

		p1, err := l.WaitForBGPEstablished(ctx, "r1", "10.203.2.12", liveConvergeTimeout)
		if err != nil {
			t.Fatal(err)
		}
		p2, err := l.WaitForBGPEstablished(ctx, "r2", "10.203.2.11", liveConvergeTimeout)
		if err != nil {
			t.Fatal(err)
		}

		// Wait briefly for initial OPEN/KEEPALIVE exchange to settle.
		err = WaitFor(ctx, 5*time.Second, DefaultPollInterval, func(pollCtx context.Context) (bool, error) {
			s1, err := l.BGPSummary(pollCtx, "r1")
			if err != nil {
				return false, nil
			}
			peer, ok := s1.Peer("10.203.2.12")
			if !ok {
				return false, nil
			}
			p1 = &peer
			return peer.IsEstablished() && peer.MsgRcvd >= 2 && peer.MsgSent >= 2, nil
		})
		if err != nil {
			t.Fatalf("r1 session did not settle: %v", err)
		}

		if !p1.IsPolicyBlocked() || !p2.IsPolicyBlocked() {
			t.Fatalf("expected peerState=Policy on both peers, got r1=%q r2=%q", p1.PeerState, p2.PeerState)
		}
		if p1.PfxSnt != 0 || p1.PfxRcd != 0 || p2.PfxSnt != 0 || p2.PfxRcd != 0 {
			t.Fatalf("expected 0 prefixes exchanged without policy, got r1=(rcd:%d snt:%d) r2=(rcd:%d snt:%d)",
				p1.PfxRcd, p1.PfxSnt, p2.PfxRcd, p2.PfxSnt)
		}

		adv, err := l.BGPAdvertisedRoutes(ctx, "r1", "10.203.2.12")
		if err != nil {
			t.Fatalf("r1 BGPAdvertisedRoutes() error = %v", err)
		}
		if adv.TotalPrefixCounter != 0 || len(adv.AdvertisedRoutes) != 0 {
			t.Fatalf("expected 0 advertised routes on r1, got %#v", adv)
		}

		rib2, err := l.IPRoutes(ctx, "r2")
		if err != nil {
			t.Fatalf("r2 IPRoutes() error = %v", err)
		}
		if rib2.HasInstalled("192.168.10.0/24", "bgp") {
			t.Fatalf("r2 unexpectedly installed 192.168.10.0/24 without policy")
		}
	})

	t.Run("prefix_list_only_satisfies_rfc8212_without_route_map", func(t *testing.T) {
		l := startLab(t, rt, Topology{
			Name:     "rfc8212-prefix-list-only",
			Networks: []NetworkSpec{{Name: "link", Subnet: "10.203.3.0/24"}},
			Nodes: []NodeSpec{
				{
					Name:       "r1",
					Daemons:    []string{"bgpd"},
					Interfaces: []InterfaceSpec{{Network: "link", IP: "10.203.3.11"}},
					Config: `frr defaults traditional
hostname r1
!
ip prefix-list ALLOW-OUT seq 10 permit 192.168.10.0/24
ip prefix-list ALLOW-OUT seq 20 deny any
ip prefix-list DENY-IN seq 10 deny any
!
ip route 192.168.10.0/24 blackhole
ip route 192.168.11.0/24 blackhole
!
router bgp 65001
 bgp router-id 10.203.3.11
 neighbor 10.203.3.12 remote-as 65002
 !
 address-family ipv4 unicast
  network 192.168.10.0/24
  network 192.168.11.0/24
  neighbor 10.203.3.12 prefix-list DENY-IN in
  neighbor 10.203.3.12 prefix-list ALLOW-OUT out
 exit-address-family
exit
`,
				},
				{
					Name:       "r2",
					Daemons:    []string{"bgpd"},
					Interfaces: []InterfaceSpec{{Network: "link", IP: "10.203.3.12"}},
					Config: `frr defaults traditional
hostname r2
!
ip prefix-list ALLOW-ALL seq 10 permit 0.0.0.0/0 le 32
!
ip route 192.168.20.0/24 blackhole
!
router bgp 65002
 bgp router-id 10.203.3.12
 neighbor 10.203.3.11 remote-as 65001
 !
 address-family ipv4 unicast
  network 192.168.20.0/24
  neighbor 10.203.3.11 prefix-list ALLOW-ALL in
  neighbor 10.203.3.11 prefix-list ALLOW-ALL out
 exit-address-family
exit
`,
				},
			},
		})

		// Wait for positive exchange in both directions: r2 installs 192.168.10.0/24
		// from r1, and r2 sends 192.168.20.0/24 toward r1.
		if err := l.WaitForIPRoute(ctx, "r2", "192.168.10.0/24", "bgp", liveConvergeTimeout); err != nil {
			t.Fatal(err)
		}
		err := WaitFor(ctx, liveConvergeTimeout, DefaultPollInterval, func(pollCtx context.Context) (bool, error) {
			adv2, err := l.BGPAdvertisedRoutes(pollCtx, "r2", "10.203.3.11")
			if err != nil {
				return false, nil
			}
			return adv2.HasPrefix("192.168.20.0/24"), nil
		})
		if err != nil {
			t.Fatalf("r2 did not advertise 192.168.20.0/24 to r1: %v", err)
		}

		sum1, err := l.BGPSummary(ctx, "r1")
		if err != nil {
			t.Fatalf("r1 BGPSummary() error = %v", err)
		}
		p1, ok := sum1.Peer("10.203.3.12")
		if !ok || !p1.IsEstablished() || !p1.IsPolicyOK() {
			t.Fatalf("expected r1 Established+OK with prefix-list only, got %#v", p1)
		}
		if p1.PfxSnt != 1 || p1.PfxRcd != 0 {
			t.Fatalf("expected r1 pfxSnt=1 pfxRcd=0, got pfxSnt=%d pfxRcd=%d", p1.PfxSnt, p1.PfxRcd)
		}

		adv1, err := l.BGPAdvertisedRoutes(ctx, "r1", "10.203.3.12")
		if err != nil {
			t.Fatalf("r1 BGPAdvertisedRoutes() error = %v", err)
		}
		if !adv1.HasPrefix("192.168.10.0/24") || adv1.HasPrefix("192.168.11.0/24") {
			t.Fatalf("expected only 192.168.10.0/24 advertised by r1, got %#v", adv1.AdvertisedRoutes)
		}

		rib2, err := l.IPRoutes(ctx, "r2")
		if err != nil {
			t.Fatalf("r2 IPRoutes() error = %v", err)
		}
		if !rib2.HasInstalledVia("192.168.10.0/24", "bgp", "10.203.3.11") {
			t.Fatalf("expected 192.168.10.0/24 in r2 RIB via 10.203.3.11")
		}
		if rib2.HasInstalled("192.168.11.0/24", "bgp") {
			t.Fatalf("r2 unexpectedly installed filtered prefix 192.168.11.0/24")
		}

		rib1, err := l.IPRoutes(ctx, "r1")
		if err != nil {
			t.Fatalf("r1 IPRoutes() error = %v", err)
		}
		if rib1.HasInstalled("192.168.20.0/24", "bgp") {
			t.Fatalf("r1 unexpectedly accepted 192.168.20.0/24 despite DENY-IN")
		}
	})

	t.Run("asymmetric_out_only_sends_but_keeps_peerState_Policy_and_drops_inbound", func(t *testing.T) {
		l := startLab(t, rt, Topology{
			Name:     "rfc8212-asymmetric-out",
			Networks: []NetworkSpec{{Name: "link", Subnet: "10.203.4.0/24"}},
			Nodes: []NodeSpec{
				{
					Name:       "r1",
					Daemons:    []string{"bgpd"},
					Interfaces: []InterfaceSpec{{Network: "link", IP: "10.203.4.11"}},
					Config: `frr defaults traditional
hostname r1
!
ip prefix-list ALLOW-OUT seq 10 permit 192.168.10.0/24
ip prefix-list ALLOW-OUT seq 20 deny any
!
ip route 192.168.10.0/24 blackhole
!
router bgp 65001
 bgp router-id 10.203.4.11
 neighbor 10.203.4.12 remote-as 65002
 !
 address-family ipv4 unicast
  network 192.168.10.0/24
  neighbor 10.203.4.12 prefix-list ALLOW-OUT out
 exit-address-family
exit
`,
				},
				{
					Name:       "r2",
					Daemons:    []string{"bgpd"},
					Interfaces: []InterfaceSpec{{Network: "link", IP: "10.203.4.12"}},
					Config: `frr defaults traditional
hostname r2
!
ip prefix-list ALLOW-ALL seq 10 permit 0.0.0.0/0 le 32
!
ip route 192.168.20.0/24 blackhole
!
router bgp 65002
 bgp router-id 10.203.4.12
 neighbor 10.203.4.11 remote-as 65001
 !
 address-family ipv4 unicast
  network 192.168.20.0/24
  neighbor 10.203.4.11 prefix-list ALLOW-ALL in
  neighbor 10.203.4.11 prefix-list ALLOW-ALL out
 exit-address-family
exit
`,
				},
			},
		})

		if err := l.WaitForIPRoute(ctx, "r2", "192.168.10.0/24", "bgp", liveConvergeTimeout); err != nil {
			t.Fatal(err)
		}
		err := WaitFor(ctx, liveConvergeTimeout, DefaultPollInterval, func(pollCtx context.Context) (bool, error) {
			adv2, err := l.BGPAdvertisedRoutes(pollCtx, "r2", "10.203.4.11")
			if err != nil {
				return false, nil
			}
			return adv2.HasPrefix("192.168.20.0/24"), nil
		})
		if err != nil {
			t.Fatalf("r2 did not advertise 192.168.20.0/24 to r1: %v", err)
		}

		sum1, err := l.BGPSummary(ctx, "r1")
		if err != nil {
			t.Fatalf("r1 BGPSummary() error = %v", err)
		}
		p1, ok := sum1.Peer("10.203.4.12")
		if !ok || !p1.IsEstablished() || !p1.IsPolicyBlocked() {
			t.Fatalf("expected r1 Established with peerState=Policy when missing inbound policy, got %#v", p1)
		}
		if p1.PfxSnt != 1 || p1.PfxRcd != 0 {
			t.Fatalf("expected r1 pfxSnt=1 pfxRcd=0, got pfxSnt=%d pfxRcd=%d", p1.PfxSnt, p1.PfxRcd)
		}

		rib1, err := l.IPRoutes(ctx, "r1")
		if err != nil {
			t.Fatalf("r1 IPRoutes() error = %v", err)
		}
		if rib1.HasInstalled("192.168.20.0/24", "bgp") {
			t.Fatalf("r1 unexpectedly accepted 192.168.20.0/24 without inbound policy")
		}
	})
}

// TestLiveBGPNetworkImportCheck proves FRR 10.5.0's default
// `bgp network import-check` exact-match behavior under `frr defaults traditional`:
//  1. No RIB route for 192.168.10.0/24: not valid/bestpath, not advertised.
//  2. Mismatched mask (/32 on lo for network /24): still not valid/bestpath, not advertised.
//  3. Exact match via `ip route 192.168.10.0/24 blackhole`: valid+bestpath, advertised, installed on peer.
//  4. Exact match via `interface lo` + `ip address 192.168.10.1/24`: valid+bestpath, advertised, installed on peer.
func TestLiveBGPNetworkImportCheck(t *testing.T) {
	rt := requireLiveRuntime(t)
	ctx := context.Background()

	// Each subtest declares a canary prefix 192.168.99.0/24 backed by a
	// blackhole route on r1. Waiting for r2 to install the canary proves BGP
	// origination and UPDATE propagation have completed before asserting
	// negative cases (no arbitrary sleep).
	buildR1 := func(r1IP, r2IP, extraStanza string) string {
		return fmt.Sprintf(`frr defaults traditional
hostname r1
!
ip prefix-list ALLOW-ALL seq 10 permit 0.0.0.0/0 le 32
!
%s
ip route 192.168.99.0/24 blackhole
!
router bgp 65001
 bgp router-id %s
 neighbor %s remote-as 65002
 !
 address-family ipv4 unicast
  network 192.168.10.0/24
  network 192.168.99.0/24
  neighbor %s prefix-list ALLOW-ALL in
  neighbor %s prefix-list ALLOW-ALL out
 exit-address-family
exit
`, extraStanza, r1IP, r2IP, r2IP, r2IP)
	}
	buildR2 := func(r1IP, r2IP string) string {
		return fmt.Sprintf(`frr defaults traditional
hostname r2
!
ip prefix-list ALLOW-ALL seq 10 permit 0.0.0.0/0 le 32
!
router bgp 65002
 bgp router-id %s
 neighbor %s remote-as 65001
 !
 address-family ipv4 unicast
  neighbor %s prefix-list ALLOW-ALL in
  neighbor %s prefix-list ALLOW-ALL out
 exit-address-family
exit
`, r2IP, r1IP, r1IP, r1IP)
	}

	cases := []struct {
		name          string
		subnet        string
		r1IP          string
		r2IP          string
		extraR1Config string
		wantAnnounced bool
	}{
		{
			name:          "no_rib_route_blocks_origination",
			subnet:        "10.203.5.0/24",
			r1IP:          "10.203.5.11",
			r2IP:          "10.203.5.12",
			extraR1Config: "!",
			wantAnnounced: false,
		},
		{
			name:   "mismatched_mask_lo_32_blocks_origination",
			subnet: "10.203.6.0/24",
			r1IP:   "10.203.6.11",
			r2IP:   "10.203.6.12",
			extraR1Config: `interface lo
 ip address 192.168.10.1/32
exit`,
			wantAnnounced: false,
		},
		{
			name:          "exact_match_static_blackhole_announces",
			subnet:        "10.203.7.0/24",
			r1IP:          "10.203.7.11",
			r2IP:          "10.203.7.12",
			extraR1Config: "ip route 192.168.10.0/24 blackhole",
			wantAnnounced: true,
		},
		{
			name:   "exact_match_loopback_24_announces",
			subnet: "10.203.8.0/24",
			r1IP:   "10.203.8.11",
			r2IP:   "10.203.8.12",
			extraR1Config: `interface lo
 ip address 192.168.10.1/24
exit`,
			wantAnnounced: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l := startLab(t, rt, Topology{
				Name:     "import-check-" + tc.name,
				Networks: []NetworkSpec{{Name: "link", Subnet: tc.subnet}},
				Nodes: []NodeSpec{
					{
						Name:       "r1",
						Daemons:    []string{"bgpd"},
						Interfaces: []InterfaceSpec{{Network: "link", IP: tc.r1IP}},
						Config:     buildR1(tc.r1IP, tc.r2IP, tc.extraR1Config),
					},
					{
						Name:       "r2",
						Daemons:    []string{"bgpd"},
						Interfaces: []InterfaceSpec{{Network: "link", IP: tc.r2IP}},
						Config:     buildR2(tc.r1IP, tc.r2IP),
					},
				},
			})

			// Canary proves BGP session is Established, Policy OK, and UPDATEs flowed.
			if err := l.WaitForIPRoute(ctx, "r2", "192.168.99.0/24", "bgp", liveConvergeTimeout); err != nil {
				t.Fatalf("canary route 192.168.99.0/24 did not reach r2: %v", err)
			}

			if tc.wantAnnounced {
				if err := l.WaitForIPRoute(ctx, "r2", "192.168.10.0/24", "bgp", liveConvergeTimeout); err != nil {
					t.Fatalf("expected 192.168.10.0/24 in r2 RIB: %v", err)
				}
			}

			bgp1, err := l.BGPRoutes(ctx, "r1")
			if err != nil {
				t.Fatalf("r1 BGPRoutes() error = %v", err)
			}
			if got := bgp1.HasValidBestPath("192.168.10.0/24"); got != tc.wantAnnounced {
				t.Errorf("r1 HasValidBestPath(192.168.10.0/24) = %v, want %v (paths=%#v)",
					got, tc.wantAnnounced, bgp1.Routes["192.168.10.0/24"])
			}

			adv1, err := l.BGPAdvertisedRoutes(ctx, "r1", tc.r2IP)
			if err != nil {
				t.Fatalf("r1 BGPAdvertisedRoutes() error = %v", err)
			}
			if got := adv1.HasPrefix("192.168.10.0/24"); got != tc.wantAnnounced {
				t.Errorf("r1 advertised 192.168.10.0/24 = %v, want %v (adv=%#v)",
					got, tc.wantAnnounced, adv1.AdvertisedRoutes)
			}

			rib2, err := l.IPRoutes(ctx, "r2")
			if err != nil {
				t.Fatalf("r2 IPRoutes() error = %v", err)
			}
			if got := rib2.HasInstalledVia("192.168.10.0/24", "bgp", tc.r1IP); got != tc.wantAnnounced {
				t.Errorf("r2 RIB installed 192.168.10.0/24 via %s = %v, want %v", tc.r1IP, got, tc.wantAnnounced)
			}
		})
	}
}

// TestLiveMultiNetworkTopology proves multi-network attachment (2+ --internal
// networks on a single transit router) live on Podman: r1 --net-a-- r2 --net-b-- r3.
func TestLiveMultiNetworkTopology(t *testing.T) {
	rt := requireLiveRuntime(t)
	ctx := context.Background()

	l := startLab(t, rt, Topology{
		Name: "bgp-transit-3node",
		Networks: []NetworkSpec{
			{Name: "net-a", Subnet: "10.203.10.0/24"},
			{Name: "net-b", Subnet: "10.203.11.0/24"},
		},
		Nodes: []NodeSpec{
			{
				Name:       "r1",
				Daemons:    []string{"bgpd"},
				Interfaces: []InterfaceSpec{{Network: "net-a", IP: "10.203.10.11"}},
				Config: `frr defaults traditional
hostname r1
!
ip prefix-list ALLOW-ALL seq 10 permit 0.0.0.0/0 le 32
!
ip route 192.168.10.0/24 blackhole
!
router bgp 65001
 bgp router-id 10.203.10.11
 neighbor 10.203.10.12 remote-as 65002
 !
 address-family ipv4 unicast
  network 192.168.10.0/24
  neighbor 10.203.10.12 prefix-list ALLOW-ALL in
  neighbor 10.203.10.12 prefix-list ALLOW-ALL out
 exit-address-family
exit
`,
			},
			{
				Name:    "r2",
				Daemons: []string{"bgpd"},
				Interfaces: []InterfaceSpec{
					{Network: "net-a", IP: "10.203.10.12"},
					{Network: "net-b", IP: "10.203.11.12"},
				},
				Config: `frr defaults traditional
hostname r2
!
ip prefix-list ALLOW-ALL seq 10 permit 0.0.0.0/0 le 32
!
router bgp 65002
 bgp router-id 10.203.10.12
 neighbor 10.203.10.11 remote-as 65001
 neighbor 10.203.11.13 remote-as 65003
 !
 address-family ipv4 unicast
  neighbor 10.203.10.11 prefix-list ALLOW-ALL in
  neighbor 10.203.10.11 prefix-list ALLOW-ALL out
  neighbor 10.203.11.13 prefix-list ALLOW-ALL in
  neighbor 10.203.11.13 prefix-list ALLOW-ALL out
 exit-address-family
exit
`,
			},
			{
				Name:       "r3",
				Daemons:    []string{"bgpd"},
				Interfaces: []InterfaceSpec{{Network: "net-b", IP: "10.203.11.13"}},
				Config: `frr defaults traditional
hostname r3
!
ip prefix-list ALLOW-ALL seq 10 permit 0.0.0.0/0 le 32
!
router bgp 65003
 bgp router-id 10.203.11.13
 neighbor 10.203.11.12 remote-as 65002
 !
 address-family ipv4 unicast
  neighbor 10.203.11.12 prefix-list ALLOW-ALL in
  neighbor 10.203.11.12 prefix-list ALLOW-ALL out
 exit-address-family
exit
`,
			},
		},
	})

	if err := l.WaitForIPRoute(ctx, "r3", "192.168.10.0/24", "bgp", liveConvergeTimeout); err != nil {
		t.Fatalf("r3 did not learn 192.168.10.0/24 via r2 transit: %v", err)
	}

	rib3, err := l.IPRoutes(ctx, "r3")
	if err != nil {
		t.Fatalf("r3 IPRoutes() error = %v", err)
	}
	if !rib3.HasInstalledVia("192.168.10.0/24", "bgp", "10.203.11.12") {
		t.Fatalf("expected 192.168.10.0/24 in r3 RIB via r2 (10.203.11.12), got %#v", rib3["192.168.10.0/24"])
	}

	bgp3, err := l.BGPRoutes(ctx, "r3")
	if err != nil {
		t.Fatalf("r3 BGPRoutes() error = %v", err)
	}
	best, ok := bgp3.BestPath("192.168.10.0/24")
	if !ok {
		t.Fatalf("expected valid bestpath for 192.168.10.0/24 on r3")
	}
	if best.Path != "65002 65001" {
		t.Errorf("r3 AS_PATH = %q, want \"65002 65001\"", best.Path)
	}
}
