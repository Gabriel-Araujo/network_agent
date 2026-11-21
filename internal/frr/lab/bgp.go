package lab

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// BGPSummary models the root JSON object returned by FRR 10.5.0 for
// `show bgp ipv4 unicast summary json`.
type BGPSummary struct {
	RouterID       string                    `json:"routerId"`
	AS             uint32                    `json:"as"`
	VRFID          int                       `json:"vrfId"`
	VRFName        string                    `json:"vrfName"`
	TableVersion   int                       `json:"tableVersion"`
	RIBCount       int                       `json:"ribCount"`
	RIBMemory      int                       `json:"ribMemory"`
	PeerCount      int                       `json:"peerCount"`
	Peers          map[string]BGPPeerSummary `json:"peers"`
	FailedPeers    int                       `json:"failedPeers"`
	DisplayedPeers int                       `json:"displayedPeers"`
	TotalPeers     int                       `json:"totalPeers"`
	DynamicPeers   int                       `json:"dynamicPeers"`
	BestPath       BGPBestPath               `json:"bestPath"`
}

// BGPBestPath models the bestPath section of BGPSummary.
// Note: in FRR 10.5.0, multiPathRelax is serialized as a string ("false"), not a bool.
type BGPBestPath struct {
	MultiPathRelax string `json:"multiPathRelax"`
}

// BGPPeerSummary models one peer entry in BGPSummary.Peers.
//
// Important FRR 10.5.0 behavior: State is "Established" even when RFC 8212
// (`bgp ebgp-requires-policy`) blocks route exchange; policy enforcement status
// is carried in PeerState ("Policy" vs "OK").
type BGPPeerSummary struct {
	Hostname                   string `json:"hostname"`
	DomainName                 string `json:"domainname"`
	SoftwareVersion            string `json:"softwareVersion"`
	RemoteAS                   uint32 `json:"remoteAs"`
	LocalAS                    uint32 `json:"localAs"`
	Version                    int    `json:"version"`
	MsgRcvd                    int    `json:"msgRcvd"`
	MsgSent                    int    `json:"msgSent"`
	TableVersion               int    `json:"tableVersion"`
	OutQ                       int    `json:"outq"`
	InQ                        int    `json:"inq"`
	PeerUptime                 string `json:"peerUptime"`
	PeerUptimeMsec             int64  `json:"peerUptimeMsec"`
	PeerUptimeEstablishedEpoch int64  `json:"peerUptimeEstablishedEpoch"`
	PfxRcd                     int    `json:"pfxRcd"`
	PfxSnt                     int    `json:"pfxSnt"`
	State                      string `json:"state"`
	PeerState                  string `json:"peerState"`
	ConnectionsEstablished     int    `json:"connectionsEstablished"`
	ConnectionsDropped         int    `json:"connectionsDropped"`
	IDType                     string `json:"idType"`
}

// IsEstablished reports whether the BGP FSM reached Established.
func (p BGPPeerSummary) IsEstablished() bool {
	return p.State == "Established"
}

// IsPolicyOK reports whether FRR reports both directions clear of RFC 8212 policy blocks.
func (p BGPPeerSummary) IsPolicyOK() bool {
	return p.PeerState == "OK"
}

// IsPolicyBlocked reports whether FRR is blocking inbound and/or outbound updates
// due to missing eBGP policy (RFC 8212).
func (p BGPPeerSummary) IsPolicyBlocked() bool {
	return p.PeerState == "Policy"
}

// Peer returns the summary for peerIP, if present.
func (s *BGPSummary) Peer(peerIP string) (BGPPeerSummary, bool) {
	if s == nil {
		return BGPPeerSummary{}, false
	}
	p, ok := s.Peers[peerIP]
	return p, ok
}

// BGPRoutes models the JSON output of `show bgp ipv4 unicast json`.
//
// Important FRR 10.5.0 behavior: when `bgp network import-check` fails for a
// local `network` statement, the prefix still appears in Routes, but without
// Valid: true and Bestpath: true. Always use HasValidBestPath or BestPath
// rather than checking map key presence alone.
type BGPRoutes struct {
	VRFID         int                  `json:"vrfId"`
	VRFName       string               `json:"vrfName"`
	TableVersion  int                  `json:"tableVersion"`
	RouterID      string               `json:"routerId"`
	DefaultLocPrf int                  `json:"defaultLocPrf"`
	LocalAS       uint32               `json:"localAS"`
	Routes        map[string][]BGPPath `json:"routes"`
	TotalRoutes   int                  `json:"totalRoutes"`
	TotalPaths    int                  `json:"totalPaths"`
}

// BGPPath models one path entry under BGPRoutes.Routes[prefix].
type BGPPath struct {
	Valid           bool         `json:"valid"`
	Bestpath        bool         `json:"bestpath"`
	SelectionReason string       `json:"selectionReason"`
	PathFrom        string       `json:"pathFrom"`
	Prefix          string       `json:"prefix"`
	PrefixLen       int          `json:"prefixLen"`
	Network         string       `json:"network"`
	Version         int          `json:"version"`
	Metric          int          `json:"metric"`
	Weight          int          `json:"weight"`
	PeerID          string       `json:"peerId"`
	Path            string       `json:"path"`
	Origin          string       `json:"origin"`
	NextHops        []BGPNextHop `json:"nexthops"`
}

// BGPNextHop models a nexthop entry in BGPPath.NextHops.
type BGPNextHop struct {
	IP         string `json:"ip"`
	Hostname   string `json:"hostname"`
	AFI        string `json:"afi"`
	Used       bool   `json:"used"`
	Accessible bool   `json:"accessible"`
}

// HasValidBestPath reports whether prefix has at least one path marked both
// Valid and Bestpath.
func (r *BGPRoutes) HasValidBestPath(prefix string) bool {
	_, ok := r.BestPath(prefix)
	return ok
}

// BestPath returns the first path for prefix marked both Valid and Bestpath.
func (r *BGPRoutes) BestPath(prefix string) (BGPPath, bool) {
	if r == nil {
		return BGPPath{}, false
	}
	for _, p := range r.Routes[prefix] {
		if p.Valid && p.Bestpath {
			return p, true
		}
	}
	return BGPPath{}, false
}

// AdvertisedRoutes models the JSON output of
// `show bgp ipv4 unicast neighbors <ip> advertised-routes json`.
// Note: unlike BGPRoutes.Routes (map to slice), FRR serializes
// advertisedRoutes as a map from prefix to a single object.
type AdvertisedRoutes struct {
	BGPTableVersion       int                        `json:"bgpTableVersion"`
	BGPLocalRouterID      string                     `json:"bgpLocalRouterId"`
	DefaultLocPrf         int                        `json:"defaultLocPrf"`
	LocalAS               uint32                     `json:"localAS"`
	AdvertisedRoutes      map[string]AdvertisedRoute `json:"advertisedRoutes"`
	TotalPrefixCounter    int                        `json:"totalPrefixCounter"`
	FilteredPrefixCounter int                        `json:"filteredPrefixCounter"`
}

// AdvertisedRoute models one advertised prefix entry.
type AdvertisedRoute struct {
	AddrPrefix string `json:"addrPrefix"`
	PrefixLen  int    `json:"prefixLen"`
	Network    string `json:"network"`
	NextHop    string `json:"nextHop"`
	Metric     int    `json:"metric"`
	Weight     int    `json:"weight"`
	Path       string `json:"path"`
	Origin     string `json:"origin"`
	Valid      bool   `json:"valid"`
	Best       bool   `json:"best"`
}

// HasPrefix reports whether prefix is present in AdvertisedRoutes.
func (a *AdvertisedRoutes) HasPrefix(prefix string) bool {
	if a == nil {
		return false
	}
	_, ok := a.AdvertisedRoutes[prefix]
	return ok
}

// IPRoutes models the JSON output of `show ip route json`.
type IPRoutes map[string][]IPRouteEntry

// IPRouteEntry models one zebra RIB entry for a prefix.
type IPRouteEntry struct {
	Prefix       string           `json:"prefix"`
	PrefixLen    int              `json:"prefixLen"`
	Protocol     string           `json:"protocol"`
	VRFID        int              `json:"vrfId"`
	VRFName      string           `json:"vrfName"`
	Selected     bool             `json:"selected"`
	DestSelected bool             `json:"destSelected"`
	Distance     int              `json:"distance"`
	Metric       int              `json:"metric"`
	Installed    bool             `json:"installed"`
	Table        int              `json:"table"`
	Uptime       string           `json:"uptime"`
	NextHops     []IPRouteNextHop `json:"nexthops"`
}

// IPRouteNextHop models one nexthop inside IPRouteEntry.
type IPRouteNextHop struct {
	Flags             int    `json:"flags"`
	FIB               bool   `json:"fib"`
	DirectlyConnected bool   `json:"directlyConnected"`
	Unreachable       bool   `json:"unreachable"`
	Blackhole         bool   `json:"blackhole"`
	IP                string `json:"ip"`
	AFI               string `json:"afi"`
	InterfaceIndex    int    `json:"interfaceIndex"`
	InterfaceName     string `json:"interfaceName"`
	Active            bool   `json:"active"`
	Weight            int    `json:"weight"`
}

// HasInstalled reports whether prefix has a selected, installed RIB entry.
// When protocol is non-empty (e.g. "bgp", "static", "connected"), the entry's
// Protocol must also match.
func (r IPRoutes) HasInstalled(prefix, protocol string) bool {
	for _, e := range r[prefix] {
		if e.Installed && e.Selected && (protocol == "" || e.Protocol == protocol) {
			return true
		}
	}
	return false
}

// HasInstalledVia reports whether prefix has a selected, installed RIB entry
// matching protocol (if non-empty) with an active nexthop pointing to nextHopIP.
func (r IPRoutes) HasInstalledVia(prefix, protocol, nextHopIP string) bool {
	for _, e := range r[prefix] {
		if !e.Installed || !e.Selected {
			continue
		}
		if protocol != "" && e.Protocol != protocol {
			continue
		}
		for _, nh := range e.NextHops {
			if nh.Active && nh.IP == nextHopIP {
				return true
			}
		}
	}
	return false
}

// BGPSummary queries `show bgp ipv4 unicast summary json` on node.
func (l *Lab) BGPSummary(ctx context.Context, node string) (*BGPSummary, error) {
	var out BGPSummary
	if err := l.VtyshJSON(ctx, node, "show bgp ipv4 unicast summary json", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// BGPRoutes queries `show bgp ipv4 unicast json` on node.
func (l *Lab) BGPRoutes(ctx context.Context, node string) (*BGPRoutes, error) {
	var out BGPRoutes
	if err := l.VtyshJSON(ctx, node, "show bgp ipv4 unicast json", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// BGPAdvertisedRoutes queries
// `show bgp ipv4 unicast neighbors <peerIP> advertised-routes json` on node.
func (l *Lab) BGPAdvertisedRoutes(ctx context.Context, node, peerIP string) (*AdvertisedRoutes, error) {
	peerIP = strings.TrimSpace(peerIP)
	if peerIP == "" {
		return nil, fmt.Errorf("peerIP must not be empty")
	}
	var out AdvertisedRoutes
	cmd := fmt.Sprintf("show bgp ipv4 unicast neighbors %s advertised-routes json", peerIP)
	if err := l.VtyshJSON(ctx, node, cmd, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// IPRoutes queries `show ip route json` on node.
func (l *Lab) IPRoutes(ctx context.Context, node string) (IPRoutes, error) {
	var out IPRoutes
	if err := l.VtyshJSON(ctx, node, "show ip route json", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// WaitForBGPEstablished polls node's BGP summary until peerIP reaches
// Established, returning the observed BGPPeerSummary.
func (l *Lab) WaitForBGPEstablished(ctx context.Context, node, peerIP string, timeout time.Duration) (*BGPPeerSummary, error) {
	var got BGPPeerSummary
	var lastState string
	err := WaitFor(ctx, timeout, DefaultPollInterval, func(pollCtx context.Context) (bool, error) {
		sum, err := l.BGPSummary(pollCtx, node)
		if err != nil {
			lastState = err.Error()
			return false, nil
		}
		peer, ok := sum.Peer(peerIP)
		if !ok {
			lastState = fmt.Sprintf("peer %s not found in summary", peerIP)
			return false, nil
		}
		lastState = fmt.Sprintf("state=%q peerState=%q", peer.State, peer.PeerState)
		if peer.IsEstablished() {
			got = peer
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return nil, fmt.Errorf("node %q peer %s did not reach Established: %w (last: %s)", node, peerIP, err, lastState)
	}
	return &got, nil
}

// WaitForIPRoute polls node's zebra RIB until prefix is installed and selected
// with the given protocol (or any protocol when protocol is empty).
func (l *Lab) WaitForIPRoute(ctx context.Context, node, prefix, protocol string, timeout time.Duration) error {
	err := WaitFor(ctx, timeout, DefaultPollInterval, func(pollCtx context.Context) (bool, error) {
		routes, err := l.IPRoutes(pollCtx, node)
		if err != nil {
			return false, nil
		}
		return routes.HasInstalled(prefix, protocol), nil
	})
	if err != nil {
		return fmt.Errorf("node %q did not install %s route %s: %w", node, protocol, prefix, err)
	}
	return nil
}
