//go:build cli

package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const (
	// SmokeImage is the image distrobox tests use as the inner container.
	// It must be lightweight, pullable from a public registry, and known to
	// work with distrobox-init.
	SmokeImage = "docker.io/library/alpine:3.21"

	// defaultCmdTimeout caps any single distrobox/container-manager
	// invocation so a stuck test does not hang the whole suite.
	defaultCmdTimeout = 5 * time.Minute
)

// nameCounter guarantees unique container names within a single test binary
// run even when t.Name() collides (sub-tests, retries).
var nameCounter atomic.Uint64

// Harness wires together everything a test needs: the binary under test, the
// container manager used to verify side effects, and a per-test HOME so
// distrobox state does not leak across tests.
type Harness struct {
	t *testing.T

	Binary           string
	ContainerManager string
	Home             string

	createdContainers []string
}

// New builds a Harness scoped to t. The HOME directory is provisioned with
// os.MkdirTemp (not t.TempDir) because rootless podman creates files in a
// user namespace that the test process cannot directly remove — Go's
// t.TempDir cleanup would fail with "permission denied" on those files.
// Instead h.cleanup invokes `podman unshare rm -rf` (when applicable) and
// then removes the directory. Containers created via NewName are also
// removed in cleanup so failing tests do not strand state.
func New(t *testing.T) *Harness {
	t.Helper()

	binary := os.Getenv("DISTROBOX_BIN")
	if binary == "" {
		t.Fatal("DISTROBOX_BIN is not set; run via `make test-cli` or set it manually")
	}
	if _, err := os.Stat(binary); err != nil {
		t.Fatalf("distrobox binary %q not found: %v", binary, err)
	}

	cm := os.Getenv("DBX_CONTAINER_MANAGER")
	if cm == "" {
		t.Fatal("DBX_CONTAINER_MANAGER must be set to podman or docker")
	}
	if _, err := exec.LookPath(cm); err != nil {
		t.Fatalf("container manager %q not on PATH: %v", cm, err)
	}

	home, err := os.MkdirTemp("", "dbxtest-home-")
	if err != nil {
		t.Fatalf("creating per-test home: %v", err)
	}

	h := &Harness{
		t:                t,
		Binary:           binary,
		ContainerManager: cm,
		Home:             home,
	}

	t.Cleanup(h.cleanup)
	return h
}

// NewName produces a container name unique to this test that fits within
// container-manager naming rules (alphanumeric + dash + underscore).
func (h *Harness) NewName(suffix string) string {
	h.t.Helper()
	clean := nameRegexp.ReplaceAllString(h.t.Name(), "-")
	clean = strings.Trim(clean, "-_")
	n := nameCounter.Add(1)
	name := fmt.Sprintf("dbxtest-%s-%d-%s", clean, n, suffix)
	if len(name) > 60 {
		name = name[:60]
	}
	h.createdContainers = append(h.createdContainers, name)
	return name
}

var nameRegexp = regexp.MustCompile(`[^A-Za-z0-9]+`)

// Track registers a container name so the harness will clean it up even if
// the test did not allocate it through NewName (e.g., assemble manifests).
func (h *Harness) Track(name string) {
	h.createdContainers = append(h.createdContainers, name)
}

// Distrobox runs the binary under test with the given arguments and returns
// the combined output along with the exit error (nil on success).
func (h *Harness) Distrobox(args ...string) (string, error) {
	h.t.Helper()
	return h.runCommand(h.Binary, args...)
}

// MustDistrobox is the same as Distrobox but fails the test on error.
func (h *Harness) MustDistrobox(args ...string) string {
	h.t.Helper()
	out, err := h.Distrobox(args...)
	if err != nil {
		h.t.Fatalf("distrobox %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
	return out
}

// CM runs the underlying container manager directly. Tests use it for
// verification (inspect, ps, image rm) and not for setup.
func (h *Harness) CM(args ...string) (string, error) {
	h.t.Helper()
	return h.runCommand(h.ContainerManager, args...)
}

// MustCM is the same as CM but fails the test on error.
func (h *Harness) MustCM(args ...string) string {
	h.t.Helper()
	out, err := h.CM(args...)
	if err != nil {
		h.t.Fatalf("%s %s failed: %v\n%s", h.ContainerManager, strings.Join(args, " "), err, out)
	}
	return out
}

// ContainerExists returns true when the container manager knows about a
// container with the given name (running or stopped).
func (h *Harness) ContainerExists(name string) bool {
	h.t.Helper()
	out, err := h.CM("ps", "-a", "--format", "{{.Names}}")
	if err != nil {
		h.t.Fatalf("listing containers: %v\n%s", err, out)
	}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if strings.TrimSpace(line) == name {
			return true
		}
	}
	return false
}

// ContainerRunning returns true if the container is up.
func (h *Harness) ContainerRunning(name string) bool {
	h.t.Helper()
	out, err := h.CM("inspect", "--format", "{{.State.Running}}", name)
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) == "true"
}

// Inspect returns raw container manager inspect output as a string.
func (h *Harness) Inspect(name string, fmtSpec string) string {
	h.t.Helper()
	out, err := h.CM("inspect", "--format", fmtSpec, name)
	if err != nil {
		h.t.Fatalf("inspect %s failed: %v\n%s", name, err, out)
	}
	return strings.TrimSpace(out)
}

// WriteFile creates a file under the harness HOME (so tests can build
// manifests without touching the real home).
func (h *Harness) WriteFile(relPath, contents string) string {
	h.t.Helper()
	abs := filepath.Join(h.Home, relPath)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		h.t.Fatalf("mkdir %s: %v", filepath.Dir(abs), err)
	}
	if err := os.WriteFile(abs, []byte(contents), 0o644); err != nil {
		h.t.Fatalf("write %s: %v", abs, err)
	}
	return abs
}

func (h *Harness) runCommand(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultCmdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = h.env()

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()

	if ctxErr := ctx.Err(); errors.Is(ctxErr, context.DeadlineExceeded) {
		return buf.String(), fmt.Errorf("command timed out after %s: %w", defaultCmdTimeout, ctxErr)
	}
	return buf.String(), err
}

func (h *Harness) env() []string {
	// Start from the inherited environment so PATH, registry creds, and
	// container-manager runtime config stay available, then override the
	// distrobox-facing variables.
	env := os.Environ()
	override := map[string]string{
		"HOME":                  h.Home,
		"XDG_CONFIG_HOME":       filepath.Join(h.Home, ".config"),
		"XDG_DATA_HOME":         filepath.Join(h.Home, ".local", "share"),
		"XDG_CACHE_HOME":        filepath.Join(h.Home, ".cache"),
		"DBX_CONTAINER_MANAGER": h.ContainerManager,
		"DBX_NON_INTERACTIVE":   "1",
	}
	// When XDG_CONFIG_HOME points at an empty per-test directory, rootless
	// podman silently falls back to compiled-in defaults instead of reading
	// /etc/containers/{storage,containers}.conf. Pin both files explicitly
	// so the system-wide overrides in the test image (mount_program for
	// overlay-on-overlayfs, utsns=private, cgroups=enabled) actually apply.
	if h.ContainerManager == "podman" {
		override["CONTAINERS_STORAGE_CONF"] = "/etc/containers/storage.conf"
		override["CONTAINERS_CONF"] = "/etc/containers/containers.conf"
	}
	return overrideEnv(env, override)
}

func overrideEnv(env []string, overrides map[string]string) []string {
	seen := make(map[string]bool, len(overrides))
	out := make([]string, 0, len(env)+len(overrides))
	for _, kv := range env {
		key := kv
		if idx := strings.IndexByte(kv, '='); idx >= 0 {
			key = kv[:idx]
		}
		if v, ok := overrides[key]; ok {
			out = append(out, key+"="+v)
			seen[key] = true
			continue
		}
		out = append(out, kv)
	}
	for k, v := range overrides {
		if !seen[k] {
			out = append(out, k+"="+v)
		}
	}
	return out
}

func (h *Harness) cleanup() {
	// Best-effort removal so a failing test leaves no orphans behind. We do
	// not surface errors because the test has already finished by the time
	// Cleanup runs.
	for _, name := range h.createdContainers {
		_, _ = h.CM("rm", "-f", name)
	}

	// Rootless podman creates per-test storage under a user namespace; the
	// resulting files are owned by mapped UIDs the test process cannot
	// remove directly. Re-enter the namespace with `podman unshare` to
	// clean them up, then drop the directory.
	if h.ContainerManager == "podman" {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "podman", "unshare", "rm", "-rf", h.Home)
		cmd.Env = h.env()
		_ = cmd.Run()
	}
	_ = os.RemoveAll(h.Home)
}
