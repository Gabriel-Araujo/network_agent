// Package container centralizes container runtime mechanics shared by the FRR
// verification layers (internal/frr/validate and internal/frr/lab).
//
// It resolves the container CLI (podman/docker) and image, applies the minimal
// non-privileged capabilities verified on Podman/WSL2, normalizes Windows host
// paths and LF line endings, and attaches harness ownership labels.
package container

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	// DefaultImage is the FRR container image verified for this harness.
	DefaultImage = "quay.io/frrouting/frr:10.5.0"

	// EnvRuntime overrides the container CLI binary (e.g. "podman", "docker").
	EnvRuntime = "FRR_CONTAINER_RUNTIME"
	// EnvImage overrides the FRR container image reference.
	EnvImage = "FRR_IMAGE"
	// EnvLive enables live container tests when set to "1".
	EnvLive = "FRR_LIVE"

	// LabelManaged marks every container and network created by this harness so
	// orphan cleanup never touches unrelated resources on the host.
	LabelManaged = "frr-harness.managed"
	// LabelRun ties resources created within a single validation or lab run.
	LabelRun = "frr-harness.run"

	// DefaultUlimitNofile caps container file descriptors to silence FRR's
	// "FD Limit set: 1048576 is stupidly large" warning on Podman.
	DefaultUlimitNofile = "nofile=4096:4096"
)

// Capabilities lists the exact Linux capabilities needed by FRR 10.5 daemons
// without using --privileged.
var Capabilities = []string{"NET_ADMIN", "NET_RAW", "SYS_ADMIN"}

var (
	// ErrRuntimeUnavailable indicates the container CLI binary is missing or
	// its daemon/machine is unreachable.
	ErrRuntimeUnavailable = errors.New("container runtime unavailable")
	// ErrImageMissing indicates the configured FRR image is not available in
	// the local container runtime.
	ErrImageMissing = errors.New("FRR container image not found locally")
)

// ExecResult holds the captured output and exit status of a subprocess.
type ExecResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Executor runs external commands. Tests can supply a fake implementation to
// exercise command assembly and output parsing offline.
type Executor interface {
	Exec(ctx context.Context, name string, args ...string) (ExecResult, error)
}

type osExecutor struct{}

func (osExecutor) Exec(ctx context.Context, name string, args ...string) (ExecResult, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	res := ExecResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			res.ExitCode = exitErr.ExitCode()
		} else {
			res.ExitCode = -1
		}
		return res, err
	}
	return res, nil
}

// Options configures container runtime and image resolution.
type Options struct {
	// Runtime explicitly sets the container CLI binary, overriding EnvRuntime
	// and auto-detection.
	Runtime string
	// Image explicitly sets the FRR image reference, overriding EnvImage and
	// DefaultImage.
	Image string

	// Executor optionally overrides subprocess execution (primarily for tests).
	Executor Executor
	// LookPath optionally overrides exec.LookPath (primarily for tests).
	LookPath func(file string) (string, error)
	// Getenv optionally overrides os.Getenv (primarily for tests).
	Getenv func(key string) string
}

// Runtime represents a resolved container CLI and FRR image ready to execute
// harness commands.
type Runtime struct {
	Binary string
	Image  string
	exec   Executor
}

// Resolve selects the container binary and FRR image following the precedence:
// explicit Options > environment variables > auto-detection (podman, docker)
// and DefaultImage.
func Resolve(opts Options) (*Runtime, error) {
	getenv := opts.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	lookPath := opts.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	executor := opts.Executor
	if executor == nil {
		executor = osExecutor{}
	}

	image := strings.TrimSpace(opts.Image)
	if image == "" {
		image = strings.TrimSpace(getenv(EnvImage))
	}
	if image == "" {
		image = DefaultImage
	}

	explicit := strings.TrimSpace(opts.Runtime)
	if explicit == "" {
		explicit = strings.TrimSpace(getenv(EnvRuntime))
	}

	var bin string
	if explicit != "" {
		found, err := lookPath(explicit)
		if err != nil {
			return nil, fmt.Errorf("%w: %q not found: %v", ErrRuntimeUnavailable, explicit, err)
		}
		bin = found
	} else {
		for _, candidate := range []string{"podman", "docker"} {
			if found, err := lookPath(candidate); err == nil {
				bin = found
				break
			}
		}
		if bin == "" {
			return nil, fmt.Errorf("%w: neither podman nor docker found in PATH", ErrRuntimeUnavailable)
		}
	}

	return &Runtime{
		Binary: bin,
		Image:  image,
		exec:   executor,
	}, nil
}

// Exec runs a subcommand against the resolved container runtime.
func (r *Runtime) Exec(ctx context.Context, args ...string) (ExecResult, error) {
	return r.exec.Exec(ctx, r.Binary, args...)
}

// Check verifies that the container runtime daemon/machine is reachable and
// that the configured FRR image exists locally.
func (r *Runtime) Check(ctx context.Context) error {
	res, err := r.Exec(ctx, "info")
	if err != nil {
		detail := strings.TrimSpace(res.Stderr)
		if detail == "" {
			detail = strings.TrimSpace(res.Stdout)
		}
		if detail != "" {
			return fmt.Errorf("%w: %s info failed: %s", ErrRuntimeUnavailable, r.Binary, detail)
		}
		return fmt.Errorf("%w: %s info failed: %v", ErrRuntimeUnavailable, r.Binary, err)
	}

	res, err = r.Exec(ctx, "image", "inspect", r.Image)
	if err != nil {
		detail := strings.TrimSpace(res.Stderr)
		if detail == "" {
			detail = strings.TrimSpace(res.Stdout)
		}
		if detail != "" {
			return fmt.Errorf("%w: %s (%s)", ErrImageMissing, r.Image, detail)
		}
		return fmt.Errorf("%w: %s (%v)", ErrImageMissing, r.Image, err)
	}
	return nil
}

// NewRunID returns a short random hex identifier suitable for container names
// and harness run labels.
func NewRunID() string {
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "run000000000"
	}
	return hex.EncodeToString(b[:])
}

// Labels returns the standard harness --label flags for containers and networks.
func Labels(runID string) []string {
	args := []string{"--label", LabelManaged + "=true"}
	if runID != "" {
		args = append(args, "--label", LabelRun+"="+runID)
	}
	return args
}

// BaseRunArgs returns the minimal capability, ulimit, and label flags required
// to run FRR daemons inside a container without --privileged.
func BaseRunArgs(runID string) []string {
	args := make([]string, 0, len(Capabilities)+6)
	for _, capName := range Capabilities {
		args = append(args, "--cap-add="+capName)
	}
	args = append(args, "--ulimit", DefaultUlimitNofile)
	args = append(args, Labels(runID)...)
	return args
}

// NormalizeHostPath converts Windows backslashes to forward slashes so host
// paths work reliably in Podman/Docker -v bind mounts across Windows and Linux.
func NormalizeHostPath(hostPath string) string {
	return strings.ReplaceAll(hostPath, "\\", "/")
}

// BindMount formats a -v bind-mount specification with a normalized host path.
func BindMount(hostPath, containerPath string, readOnly bool) string {
	spec := NormalizeHostPath(hostPath) + ":" + containerPath
	if readOnly {
		spec += ":ro"
	}
	return spec
}
