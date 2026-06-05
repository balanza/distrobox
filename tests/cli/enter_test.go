//go:build cli

package cli

import (
	"strings"
	"testing"
)

// createEnterContainer creates a fresh distrobox container using the smoke
// image so each test in this file starts from a known-good state.
func createEnterContainer(t *testing.T, h *Harness, name string) {
	t.Helper()
	h.MustDistrobox("create", "--yes", "--image", SmokeImage, "--name", name)
}

// TestEnter_RunsCommand verifies that `distrobox enter --no-tty <name> -- echo hello`
// executes the command inside the container and returns its stdout.
func TestEnter_RunsCommand(t *testing.T) {
	h := New(t)
	name := h.NewName("runs-command")
	createEnterContainer(t, h, name)

	out := h.MustDistrobox("enter", "--no-tty", "--name", name, "--", "echo", "hello")
	if !strings.Contains(out, "hello") {
		t.Fatalf("expected enter output to contain %q, got: %q", "hello", out)
	}
}

// TestEnter_PositionalName confirms that the container name can be supplied
// positionally (no --name flag) before the `--` separator.
func TestEnter_PositionalName(t *testing.T) {
	h := New(t)
	name := h.NewName("positional")
	createEnterContainer(t, h, name)

	out := h.MustDistrobox("enter", "--no-tty", name, "--", "echo", "hello")
	if !strings.Contains(out, "hello") {
		t.Fatalf("expected enter output to contain %q, got: %q", "hello", out)
	}
}

// TestEnter_ExitCodePropagates ensures that a non-zero exit from the command
// run inside the container surfaces as a non-nil error from distrobox enter.
func TestEnter_ExitCodePropagates(t *testing.T) {
	h := New(t)
	name := h.NewName("exit-code")
	createEnterContainer(t, h, name)

	out, err := h.Distrobox("enter", "--no-tty", "--name", name, "--", "sh", "-c", "exit 7")
	if err == nil {
		t.Fatalf("expected non-nil error when inner command exits 7, got nil\noutput: %s", out)
	}
}

// TestEnter_DryRunPrintsCommand checks that --dry-run prints the underlying
// container-manager command (containing e.g. "podman" or "docker") instead of
// actually entering the container.
func TestEnter_DryRunPrintsCommand(t *testing.T) {
	h := New(t)
	name := h.NewName("dry-run")
	createEnterContainer(t, h, name)

	out := h.MustDistrobox("enter", "--dry-run", "--name", name)
	if !strings.Contains(out, h.ContainerManager) {
		t.Fatalf("expected dry-run output to mention container manager %q, got: %q", h.ContainerManager, out)
	}
}

// TestEnter_PassesAdditionalFlags verifies that --additional-flags are forwarded
// to the container manager exec invocation, so e.g. extra env vars reach the
// inner command.
func TestEnter_PassesAdditionalFlags(t *testing.T) {
	h := New(t)
	name := h.NewName("addl-flags")
	createEnterContainer(t, h, name)

	out := h.MustDistrobox(
		"enter",
		"--no-tty",
		"--name", name,
		"--additional-flags", "--env FOO=bar",
		"--",
		"sh", "-c", "echo $FOO",
	)
	if !strings.Contains(out, "bar") {
		t.Fatalf("expected enter output to contain %q (from --additional-flags env), got: %q", "bar", out)
	}
}
