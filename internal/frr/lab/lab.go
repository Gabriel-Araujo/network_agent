// Package lab implements Layer 2 of the FRR verification harness:
// multi-container topology proofs (Prova de topologia in CONTEXT.md).
//
// It boots two or more FRR containers connected across one or more isolated
// (--internal) container networks with static IPs, queries live control-plane
// and RIB state via vtysh JSON commands, and guarantees portable two-stage
// teardown even on partial startup failure or test cancellation.
package lab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Gabriel-Araujo/network_agent/internal/frr/container"
)

const (
	// DefaultReadyTimeout caps waiting for FRR daemons inside each node to
	// become reachable via vtysh during Start.
	DefaultReadyTimeout = 30 * time.Second
	// DefaultPollInterval is the polling cadence used by WaitFor and readiness
	// checks when no interval is specified.
	DefaultPollInterval = 150 * time.Millisecond
	// DefaultTeardownTimeout bounds container and network removal in Close(),
	// using a fresh background context so cleanup survives test cancellation.
	DefaultTeardownTimeout = 30 * time.Second
)

// Topology declares an isolated multi-container FRR testbed.
type Topology struct {
	// Name identifies the topology for diagnostics (optional).
	Name string
	// Networks lists the isolated (--internal) container networks to create.
	Networks []NetworkSpec
	// Nodes lists the FRR containers to boot and attach to Networks.
	Nodes []NodeSpec
	// Runtime configures container runtime/image resolution when using Start.
	Runtime container.Options
	// ReadyTimeout overrides DefaultReadyTimeout for waiting on node daemons.
	ReadyTimeout time.Duration
	// ReadyPollInterval overrides DefaultPollInterval during readiness polling.
	ReadyPollInterval time.Duration
}

// NetworkSpec declares one isolated (--internal) bridge network.
type NetworkSpec struct {
	// Name is the logical network name referenced by InterfaceSpec.Network.
	Name string
	// Subnet is the CIDR subnet (e.g. "10.202.0.0/24") assigned to the network.
	Subnet string
}

// NodeSpec declares one FRR router container in the topology.
type NodeSpec struct {
	// Name is the logical node name (also used as container hostname).
	Name string
	// Daemons lists the optional FRR daemons to enable in /etc/frr/daemons
	// (e.g. []string{"bgpd"}). When nil, container.AllDaemons are enabled; an
	// explicit non-nil empty slice enables only the always-on core daemons
	// (mgmtd, zebra, staticd).
	Daemons []string
	// Config is the full /etc/frr/frr.conf mounted into the container.
	Config string
	// Interfaces attaches this node to one or more declared networks.
	Interfaces []InterfaceSpec
}

// InterfaceSpec attaches a node to a declared network, optionally with a
// static IPv4/IPv6 address inside that network's subnet.
type InterfaceSpec struct {
	Network string
	IP      string
}

// Lab represents a running multi-container FRR topology. Call Close when done
// (typically via defer or t.Cleanup).
type Lab struct {
	rt            *container.Runtime
	runID         string
	tmpDir        string
	networks      map[string]string // logical name -> container runtime network name
	networkOrder  []string          // runtime network names in creation order
	nodes         map[string]string // logical name -> container runtime container name
	nodeOrder     []string          // runtime container names in creation order
	teardownLimit time.Duration

	closeMu sync.Mutex
	closed  bool
}

// Start resolves the container runtime, validates the topology, creates its
// isolated networks and FRR containers, and waits until all nodes' daemons are
// reachable via vtysh. On any error, all partially created resources are torn
// down before returning.
func Start(ctx context.Context, topo Topology) (*Lab, error) {
	rt, err := container.Resolve(topo.Runtime)
	if err != nil {
		return nil, err
	}
	if err := rt.Check(ctx); err != nil {
		return nil, err
	}
	return StartWithRuntime(ctx, rt, topo)
}

// StartWithRuntime boots a Topology using an already-resolved container
// Runtime. On any failure during startup or readiness waiting, all resources
// created so far are removed before returning the error.
func StartWithRuntime(ctx context.Context, rt *container.Runtime, topo Topology) (*Lab, error) {
	if rt == nil {
		return nil, errors.New("container runtime is nil")
	}
	if err := validateTopology(topo); err != nil {
		return nil, err
	}

	tmpDir, err := os.MkdirTemp("", "frr-lab-*")
	if err != nil {
		return nil, fmt.Errorf("create lab temp dir: %w", err)
	}

	runID := container.NewRunID()
	l := &Lab{
		rt:            rt,
		runID:         runID,
		tmpDir:        tmpDir,
		networks:      make(map[string]string, len(topo.Networks)),
		nodes:         make(map[string]string, len(topo.Nodes)),
		teardownLimit: DefaultTeardownTimeout,
	}

	ok := false
	defer func() {
		if !ok {
			_ = l.Close()
		}
	}()

	for _, netSpec := range topo.Networks {
		rtNetName := fmt.Sprintf("frr-lab-%s-%s", runID, netSpec.Name)
		args := buildNetworkCreateArgs(rtNetName, netSpec.Subnet, runID)
		res, err := rt.Exec(ctx, args...)
		if err != nil {
			return nil, fmt.Errorf("create network %q (%s): %w", netSpec.Name, execDetail(res, err), err)
		}
		l.networks[netSpec.Name] = rtNetName
		l.networkOrder = append(l.networkOrder, rtNetName)
	}

	expectedDaemons := make(map[string][]string, len(topo.Nodes))
	for _, nodeSpec := range topo.Nodes {
		nodeDir := filepath.Join(tmpDir, nodeSpec.Name)
		if err := os.MkdirAll(nodeDir, 0o755); err != nil {
			return nil, fmt.Errorf("create node dir for %q: %w", nodeSpec.Name, err)
		}

		daemons := nodeSpec.Daemons
		if daemons == nil {
			daemons = container.AllDaemons
		}
		expectedDaemons[nodeSpec.Name] = daemons

		files, err := writeNodeFiles(nodeDir, daemons, nodeSpec.Config)
		if err != nil {
			return nil, fmt.Errorf("prepare files for node %q: %w", nodeSpec.Name, err)
		}

		cname := fmt.Sprintf("frr-lab-%s-%s", runID, nodeSpec.Name)
		primary := nodeSpec.Interfaces[0]
		createArgs := buildContainerCreateArgs(cname, nodeSpec.Name, rt.Image, runID, l.networks[primary.Network], primary.IP, files)
		res, err := rt.Exec(ctx, createArgs...)
		if err != nil {
			return nil, fmt.Errorf("create container for node %q (%s): %w", nodeSpec.Name, execDetail(res, err), err)
		}
		l.nodes[nodeSpec.Name] = cname
		l.nodeOrder = append(l.nodeOrder, cname)

		for _, iface := range nodeSpec.Interfaces[1:] {
			connectArgs := buildNetworkConnectArgs(l.networks[iface.Network], iface.IP, cname)
			res, err := rt.Exec(ctx, connectArgs...)
			if err != nil {
				return nil, fmt.Errorf("connect node %q to network %q (%s): %w", nodeSpec.Name, iface.Network, execDetail(res, err), err)
			}
		}

		res, err = rt.Exec(ctx, "start", cname)
		if err != nil {
			return nil, fmt.Errorf("start container for node %q (%s): %w", nodeSpec.Name, execDetail(res, err), err)
		}
	}

	readyTimeout := topo.ReadyTimeout
	if readyTimeout <= 0 {
		readyTimeout = DefaultReadyTimeout
	}
	pollInterval := topo.ReadyPollInterval
	if pollInterval <= 0 {
		pollInterval = DefaultPollInterval
	}

	for _, nodeSpec := range topo.Nodes {
		if err := l.waitNodeReady(ctx, nodeSpec.Name, expectedDaemons[nodeSpec.Name], readyTimeout, pollInterval); err != nil {
			logs := l.captureLogs(nodeSpec.Name)
			if logs != "" {
				return nil, fmt.Errorf("node %q did not become ready: %w\ncontainer logs:\n%s", nodeSpec.Name, err, logs)
			}
			return nil, fmt.Errorf("node %q did not become ready: %w", nodeSpec.Name, err)
		}
	}

	ok = true
	return l, nil
}

// RunID returns the unique harness run identifier attached via LabelRun to all
// containers and networks in this Lab.
func (l *Lab) RunID() string {
	return l.runID
}

// ContainerName returns the runtime container name for a logical node.
func (l *Lab) ContainerName(node string) (string, bool) {
	name, ok := l.nodes[node]
	return name, ok
}

// NetworkName returns the runtime network name for a logical network.
func (l *Lab) NetworkName(network string) (string, bool) {
	name, ok := l.networks[network]
	return name, ok
}

// Exec runs an arbitrary command inside a node's container.
func (l *Lab) Exec(ctx context.Context, node string, args ...string) (container.ExecResult, error) {
	cname, ok := l.nodes[node]
	if !ok {
		return container.ExecResult{}, fmt.Errorf("unknown lab node %q", node)
	}
	if len(args) == 0 {
		return container.ExecResult{}, errors.New("exec requires at least one argument")
	}
	cmdArgs := make([]string, 0, len(args)+2)
	cmdArgs = append(cmdArgs, "exec", cname)
	cmdArgs = append(cmdArgs, args...)
	return l.rt.Exec(ctx, cmdArgs...)
}

// Vtysh runs one or more vtysh commands (`vtysh -c <cmd>...`) inside a node's
// container and returns stdout.
func (l *Lab) Vtysh(ctx context.Context, node string, cmds ...string) (string, error) {
	if len(cmds) == 0 {
		return "", errors.New("vtysh requires at least one command")
	}
	args := make([]string, 0, 1+2*len(cmds))
	args = append(args, "vtysh")
	for _, cmd := range cmds {
		args = append(args, "-c", cmd)
	}
	res, err := l.Exec(ctx, node, args...)
	if err != nil {
		return res.Stdout, fmt.Errorf("node %q vtysh %q failed (exit %d): %s", node, strings.Join(cmds, "; "), res.ExitCode, execDetail(res, err))
	}
	return res.Stdout, nil
}

// VtyshJSON runs a vtysh JSON command inside a node's container and decodes the
// JSON output into dst.
func (l *Lab) VtyshJSON(ctx context.Context, node, cmd string, dst any) error {
	out, err := l.Vtysh(ctx, node, cmd)
	if err != nil {
		return err
	}
	trimmed := strings.TrimSpace(out)
	if err := json.Unmarshal([]byte(trimmed), dst); err != nil {
		return fmt.Errorf("node %q decode JSON from %q: %w (raw=%q)", node, cmd, err, truncateRaw(trimmed, 400))
	}
	return nil
}

// Close tears down all containers, networks, and temporary config files created
// by this Lab using a portable two-stage sequence (`rm -f -t 0` followed by
// `network rm`) under an independent background context so cleanup succeeds
// even when the test context has been cancelled or expired. Close is safe to
// call multiple times.
func (l *Lab) Close() error {
	if l == nil {
		return nil
	}
	l.closeMu.Lock()
	if l.closed {
		l.closeMu.Unlock()
		return nil
	}
	l.closed = true
	l.closeMu.Unlock()

	limit := l.teardownLimit
	if limit <= 0 {
		limit = DefaultTeardownTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()

	var errs []error
	if len(l.nodeOrder) > 0 {
		args := make([]string, 0, len(l.nodeOrder)+4)
		args = append(args, "rm", "-f", "-t", "0")
		args = append(args, l.nodeOrder...)
		if res, err := l.rt.Exec(ctx, args...); err != nil {
			errs = append(errs, fmt.Errorf("remove lab containers: %s (%w)", execDetail(res, err), err))
		}
	}

	if len(l.networkOrder) > 0 {
		args := make([]string, 0, len(l.networkOrder)+2)
		args = append(args, "network", "rm")
		args = append(args, l.networkOrder...)
		if res, err := l.rt.Exec(ctx, args...); err != nil {
			errs = append(errs, fmt.Errorf("remove lab networks: %s (%w)", execDetail(res, err), err))
		}
	}

	if l.tmpDir != "" {
		if err := os.RemoveAll(l.tmpDir); err != nil {
			errs = append(errs, fmt.Errorf("remove lab temp dir %q: %w", l.tmpDir, err))
		}
	}

	return errors.Join(errs...)
}

// SweepOrphans resolves the container runtime and removes any leftover
// containers and networks carrying label frr-harness.managed=true.
func SweepOrphans(ctx context.Context, opts container.Options) error {
	rt, err := container.Resolve(opts)
	if err != nil {
		return err
	}
	return SweepOrphansWithRuntime(ctx, rt)
}

// SweepOrphansWithRuntime removes all containers and networks carrying label
// frr-harness.managed=true in two portable stages (`rm -f -t 0` then
// `network rm`), leaving unrelated host containers and networks untouched.
func SweepOrphansWithRuntime(ctx context.Context, rt *container.Runtime) error {
	if rt == nil {
		return errors.New("container runtime is nil")
	}
	filter := "label=" + container.LabelManaged + "=true"
	var errs []error

	psRes, err := rt.Exec(ctx, "ps", "-aq", "--filter", filter)
	if err != nil {
		errs = append(errs, fmt.Errorf("list orphan containers: %s (%w)", execDetail(psRes, err), err))
	} else if ids := parseIDLines(psRes.Stdout); len(ids) > 0 {
		rmArgs := make([]string, 0, len(ids)+4)
		rmArgs = append(rmArgs, "rm", "-f", "-t", "0")
		rmArgs = append(rmArgs, ids...)
		if res, err := rt.Exec(ctx, rmArgs...); err != nil {
			errs = append(errs, fmt.Errorf("remove orphan containers: %s (%w)", execDetail(res, err), err))
		}
	}

	netRes, err := rt.Exec(ctx, "network", "ls", "-q", "--filter", filter)
	if err != nil {
		errs = append(errs, fmt.Errorf("list orphan networks: %s (%w)", execDetail(netRes, err), err))
	} else if ids := parseIDLines(netRes.Stdout); len(ids) > 0 {
		rmArgs := make([]string, 0, len(ids)+2)
		rmArgs = append(rmArgs, "network", "rm")
		rmArgs = append(rmArgs, ids...)
		if res, err := rt.Exec(ctx, rmArgs...); err != nil {
			errs = append(errs, fmt.Errorf("remove orphan networks: %s (%w)", execDetail(res, err), err))
		}
	}

	return errors.Join(errs...)
}

// WaitFor polls fn until it returns (true, nil), returns a non-nil error,
// timeout elapses, or ctx is cancelled. fn is invoked once immediately before
// waiting for interval.
func WaitFor(ctx context.Context, timeout, interval time.Duration, fn func(ctx context.Context) (bool, error)) error {
	if fn == nil {
		return errors.New("WaitFor condition function is nil")
	}
	if interval <= 0 {
		interval = DefaultPollInterval
	}

	waitCtx := ctx
	var cancel context.CancelFunc
	if timeout > 0 {
		waitCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	ok, err := fn(waitCtx)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-waitCtx.Done():
			if timeout > 0 && errors.Is(waitCtx.Err(), context.DeadlineExceeded) && ctx.Err() == nil {
				return fmt.Errorf("condition not met within %s: %w", timeout, context.DeadlineExceeded)
			}
			return waitCtx.Err()
		case <-ticker.C:
			ok, err := fn(waitCtx)
			if err != nil {
				return err
			}
			if ok {
				return nil
			}
		}
	}
}

func (l *Lab) waitNodeReady(ctx context.Context, node string, optionalDaemons []string, timeout, interval time.Duration) error {
	required := make([]string, 0, len(optionalDaemons)+3)
	required = append(required, "mgmtd", "zebra", "staticd")
	for _, d := range optionalDaemons {
		d = strings.TrimSpace(d)
		if d != "" {
			required = append(required, d)
		}
	}

	var lastDetail string
	err := WaitFor(ctx, timeout, interval, func(pollCtx context.Context) (bool, error) {
		out, err := l.Vtysh(pollCtx, node, "show daemons")
		if err != nil {
			lastDetail = err.Error()
			return false, nil
		}
		running := make(map[string]bool)
		for _, tok := range strings.Fields(out) {
			running[tok] = true
		}
		for _, want := range required {
			if !running[want] {
				lastDetail = fmt.Sprintf("daemon %q not listed in show daemons (%q)", want, strings.TrimSpace(out))
				return false, nil
			}
		}
		return true, nil
	})
	if err != nil && lastDetail != "" {
		return fmt.Errorf("%w (last check: %s)", err, lastDetail)
	}
	return err
}

func (l *Lab) captureLogs(node string) string {
	cname, ok := l.nodes[node]
	if !ok {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, _ := l.rt.Exec(ctx, "logs", cname)
	combined := strings.TrimSpace(res.Stdout + "\n" + res.Stderr)
	return truncateRaw(combined, 1500)
}

type nodeFiles struct {
	daemons   string
	vtyshConf string
	frrConf   string
}

func writeNodeFiles(dir string, daemons []string, configText string) (nodeFiles, error) {
	nf := nodeFiles{
		daemons:   filepath.Join(dir, "daemons"),
		vtyshConf: filepath.Join(dir, "vtysh.conf"),
		frrConf:   filepath.Join(dir, "frr.conf"),
	}
	if err := container.WriteLF(nf.daemons, container.RenderDaemons(daemons), 0o644); err != nil {
		return nodeFiles{}, err
	}
	if err := container.WriteLF(nf.vtyshConf, container.DefaultVtyshConf, 0o644); err != nil {
		return nodeFiles{}, err
	}
	normalized := container.NormalizeLF(configText)
	if normalized != "" && !strings.HasSuffix(normalized, "\n") {
		normalized += "\n"
	}
	if err := container.WriteLF(nf.frrConf, normalized, 0o644); err != nil {
		return nodeFiles{}, err
	}
	return nf, nil
}

func buildNetworkCreateArgs(rtNetName, subnet, runID string) []string {
	args := []string{"network", "create", "--internal", "--subnet", subnet}
	args = append(args, container.Labels(runID)...)
	args = append(args, rtNetName)
	return args
}

func buildContainerCreateArgs(cname, hostname, image, runID, primaryNet, primaryIP string, nf nodeFiles) []string {
	args := []string{"create", "--name", cname, "--hostname", hostname}
	args = append(args, container.BaseRunArgs(runID)...)
	args = append(args, "--network", primaryNet)
	if primaryIP != "" {
		args = append(args, "--ip", primaryIP)
	}
	args = append(args,
		"-v", container.BindMount(nf.daemons, "/etc/frr/daemons", true),
		"-v", container.BindMount(nf.vtyshConf, "/etc/frr/vtysh.conf", true),
		"-v", container.BindMount(nf.frrConf, "/etc/frr/frr.conf", true),
		image,
	)
	return args
}

func buildNetworkConnectArgs(rtNetName, ip, cname string) []string {
	args := []string{"network", "connect"}
	if ip != "" {
		args = append(args, "--ip", ip)
	}
	args = append(args, rtNetName, cname)
	return args
}

func validateTopology(topo Topology) error {
	if len(topo.Networks) == 0 {
		return errors.New("topology must declare at least one network")
	}
	if len(topo.Nodes) == 0 {
		return errors.New("topology must declare at least one node")
	}

	netPrefixes := make(map[string]netip.Prefix, len(topo.Networks))
	usedIPs := make(map[string]map[netip.Addr]string, len(topo.Networks))
	for _, n := range topo.Networks {
		if !isSafeName(n.Name) {
			return fmt.Errorf("invalid network name %q", n.Name)
		}
		if _, dup := netPrefixes[n.Name]; dup {
			return fmt.Errorf("duplicate network name %q", n.Name)
		}
		prefix, err := netip.ParsePrefix(strings.TrimSpace(n.Subnet))
		if err != nil {
			return fmt.Errorf("network %q has invalid subnet %q: %w", n.Name, n.Subnet, err)
		}
		netPrefixes[n.Name] = prefix
		usedIPs[n.Name] = make(map[netip.Addr]string)
	}

	seenNodes := make(map[string]bool, len(topo.Nodes))
	for _, node := range topo.Nodes {
		if !isSafeName(node.Name) {
			return fmt.Errorf("invalid node name %q", node.Name)
		}
		if seenNodes[node.Name] {
			return fmt.Errorf("duplicate node name %q", node.Name)
		}
		seenNodes[node.Name] = true

		if len(node.Interfaces) == 0 {
			return fmt.Errorf("node %q must declare at least one interface", node.Name)
		}
		seenNets := make(map[string]bool, len(node.Interfaces))
		for _, iface := range node.Interfaces {
			prefix, exists := netPrefixes[iface.Network]
			if !exists {
				return fmt.Errorf("node %q references unknown network %q", node.Name, iface.Network)
			}
			if seenNets[iface.Network] {
				return fmt.Errorf("node %q attaches to network %q more than once", node.Name, iface.Network)
			}
			seenNets[iface.Network] = true

			ipStr := strings.TrimSpace(iface.IP)
			if ipStr != "" {
				addr, err := netip.ParseAddr(ipStr)
				if err != nil {
					return fmt.Errorf("node %q has invalid IP %q on network %q: %w", node.Name, iface.IP, iface.Network, err)
				}
				if !prefix.Contains(addr) {
					return fmt.Errorf("node %q IP %s is outside network %q subnet %s", node.Name, addr, iface.Network, prefix)
				}
				if prevNode, clash := usedIPs[iface.Network][addr]; clash {
					return fmt.Errorf("IP %s on network %q assigned to both %q and %q", addr, iface.Network, prevNode, node.Name)
				}
				usedIPs[iface.Network][addr] = node.Name
			}
		}
	}
	return nil
}

func isSafeName(name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		alnum := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
		if i == 0 && !alnum {
			return false
		}
		if !alnum && c != '-' && c != '_' && c != '.' {
			return false
		}
	}
	return true
}

func parseIDLines(stdout string) []string {
	var ids []string
	for _, line := range strings.Split(stdout, "\n") {
		if id := strings.TrimSpace(line); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

func execDetail(res container.ExecResult, err error) string {
	if s := strings.TrimSpace(res.Stderr); s != "" {
		return s
	}
	if s := strings.TrimSpace(res.Stdout); s != "" {
		return s
	}
	if err != nil {
		return err.Error()
	}
	return "unknown error"
}

func truncateRaw(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "...(truncated)"
}
