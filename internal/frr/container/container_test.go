package container

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

type fakeExecutor struct {
	fn func(ctx context.Context, name string, args ...string) (ExecResult, error)
}

func (f fakeExecutor) Exec(ctx context.Context, name string, args ...string) (ExecResult, error) {
	return f.fn(ctx, name, args...)
}

func TestResolvePrecedence(t *testing.T) {
	env := map[string]string{
		EnvRuntime: "docker",
		EnvImage:   "custom/frr:env",
	}
	lookPath := func(file string) (string, error) {
		if file == "podman" || file == "docker" || file == "/custom/bin/podman" {
			return "/resolved/" + file, nil
		}
		return "", errors.New("not found")
	}

	t.Run("explicit options override env and defaults", func(t *testing.T) {
		rt, err := Resolve(Options{
			Runtime:  "/custom/bin/podman",
			Image:    "custom/frr:explicit",
			Getenv:   func(k string) string { return env[k] },
			LookPath: lookPath,
		})
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if rt.Binary != "/resolved//custom/bin/podman" {
			t.Errorf("Binary = %q, want /resolved//custom/bin/podman", rt.Binary)
		}
		if rt.Image != "custom/frr:explicit" {
			t.Errorf("Image = %q, want custom/frr:explicit", rt.Image)
		}
	})

	t.Run("env overrides auto-detection and default image", func(t *testing.T) {
		rt, err := Resolve(Options{
			Getenv:   func(k string) string { return env[k] },
			LookPath: lookPath,
		})
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if rt.Binary != "/resolved/docker" {
			t.Errorf("Binary = %q, want /resolved/docker", rt.Binary)
		}
		if rt.Image != "custom/frr:env" {
			t.Errorf("Image = %q, want custom/frr:env", rt.Image)
		}
	})

	t.Run("auto-detects podman before docker with default image", func(t *testing.T) {
		rt, err := Resolve(Options{
			Getenv:   func(string) string { return "" },
			LookPath: lookPath,
		})
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if rt.Binary != "/resolved/podman" {
			t.Errorf("Binary = %q, want /resolved/podman", rt.Binary)
		}
		if rt.Image != DefaultImage {
			t.Errorf("Image = %q, want %q", rt.Image, DefaultImage)
		}
	})

	t.Run("falls back to docker when podman missing", func(t *testing.T) {
		rt, err := Resolve(Options{
			Getenv: func(string) string { return "" },
			LookPath: func(file string) (string, error) {
				if file == "docker" {
					return "/usr/bin/docker", nil
				}
				return "", errors.New("not found")
			},
		})
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if rt.Binary != "/usr/bin/docker" {
			t.Errorf("Binary = %q, want /usr/bin/docker", rt.Binary)
		}
	})

	t.Run("returns ErrRuntimeUnavailable when none found", func(t *testing.T) {
		_, err := Resolve(Options{
			Getenv:   func(string) string { return "" },
			LookPath: func(string) (string, error) { return "", errors.New("missing") },
		})
		if !errors.Is(err, ErrRuntimeUnavailable) {
			t.Fatalf("err = %v, want ErrRuntimeUnavailable", err)
		}
	})
}

func TestRuntimeCheck(t *testing.T) {
	t.Run("distinguishes daemon unreachable from missing image", func(t *testing.T) {
		rt := &Runtime{
			Binary: "podman",
			Image:  DefaultImage,
			exec: fakeExecutor{fn: func(_ context.Context, _ string, args ...string) (ExecResult, error) {
				if len(args) > 0 && args[0] == "info" {
					return ExecResult{Stderr: "cannot connect to Podman machine", ExitCode: 125}, errors.New("exit 125")
				}
				return ExecResult{}, nil
			}},
		}
		if err := rt.Check(context.Background()); !errors.Is(err, ErrRuntimeUnavailable) {
			t.Fatalf("Check() err = %v, want ErrRuntimeUnavailable", err)
		}
	})

	t.Run("reports ErrImageMissing when image inspect fails", func(t *testing.T) {
		rt := &Runtime{
			Binary: "podman",
			Image:  DefaultImage,
			exec: fakeExecutor{fn: func(_ context.Context, _ string, args ...string) (ExecResult, error) {
				if len(args) >= 2 && args[0] == "image" && args[1] == "inspect" {
					return ExecResult{Stderr: "image not known", ExitCode: 125}, errors.New("exit 125")
				}
				return ExecResult{}, nil
			}},
		}
		if err := rt.Check(context.Background()); !errors.Is(err, ErrImageMissing) {
			t.Fatalf("Check() err = %v, want ErrImageMissing", err)
		}
	})

	t.Run("succeeds when info and image inspect pass", func(t *testing.T) {
		var calls [][]string
		rt := &Runtime{
			Binary: "podman",
			Image:  DefaultImage,
			exec: fakeExecutor{fn: func(_ context.Context, _ string, args ...string) (ExecResult, error) {
				calls = append(calls, append([]string(nil), args...))
				return ExecResult{}, nil
			}},
		}
		if err := rt.Check(context.Background()); err != nil {
			t.Fatalf("Check() unexpected error: %v", err)
		}
		if len(calls) != 2 || calls[0][0] != "info" || !slices.Equal(calls[1], []string{"image", "inspect", DefaultImage}) {
			t.Fatalf("unexpected exec calls: %#v", calls)
		}
	})
}

func TestBaseRunArgsAndBindMount(t *testing.T) {
	args := BaseRunArgs("abc123")
	joined := strings.Join(args, " ")
	for _, want := range []string{
		"--cap-add=NET_ADMIN",
		"--cap-add=NET_RAW",
		"--cap-add=SYS_ADMIN",
		"--ulimit nofile=4096:4096",
		"--label frr-harness.managed=true",
		"--label frr-harness.run=abc123",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("BaseRunArgs missing %q in %q", want, joined)
		}
	}
	if strings.Contains(joined, "--privileged") {
		t.Errorf("BaseRunArgs must never include --privileged: %q", joined)
	}

	gotRO := BindMount(`C:\Users\Gabriel\AppData\Local\Temp\frr.conf`, "/tmp/check.conf", true)
	wantRO := "C:/Users/Gabriel/AppData/Local/Temp/frr.conf:/tmp/check.conf:ro"
	if gotRO != wantRO {
		t.Errorf("BindMount(ro) = %q, want %q", gotRO, wantRO)
	}
	gotRW := BindMount(`C:\tmp\a`, "/etc/frr/a", false)
	if gotRW != "C:/tmp/a:/etc/frr/a" {
		t.Errorf("BindMount(rw) = %q, want C:/tmp/a:/etc/frr/a", gotRW)
	}
}

func TestRenderDaemonsAndWriteLF(t *testing.T) {
	all := RenderDaemons(AllDaemons)
	if strings.Contains(all, "\r") {
		t.Fatal("RenderDaemons must use LF only")
	}
	for _, d := range AllDaemons {
		if !strings.Contains(all, d+"=yes\n") {
			t.Errorf("expected %s=yes in rendered daemons", d)
		}
	}
	if !strings.Contains(all, "sharpd=no\n") {
		t.Errorf("sharpd must stay disabled by default")
	}
	if !strings.Contains(all, `bgpd_options="   -A 127.0.0.1 -M rpki -M bmp"`) {
		t.Errorf("bgpd_options must include -M rpki -M bmp")
	}
	if !strings.Contains(all, "MAX_FDS=4096\n") {
		t.Errorf("MAX_FDS=4096 missing")
	}

	bgpOnly := RenderDaemons([]string{"bgpd"})
	if !strings.Contains(bgpOnly, "bgpd=yes\n") || !strings.Contains(bgpOnly, "ospfd=no\n") {
		t.Errorf("unexpected bgpOnly daemons output:\n%s", bgpOnly)
	}

	p := filepath.Join(t.TempDir(), "test.conf")
	if err := WriteLF(p, "line1\r\nline2\rline3\n", 0o644); err != nil {
		t.Fatalf("WriteLF() error = %v", err)
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(raw) != "line1\nline2\nline3\n" {
		t.Fatalf("file content = %q, want LF-only", string(raw))
	}
}
