//go:build cli

package cli

import (
	"testing"
)

// createStopContainer creates a fresh distrobox container and starts it via
// the container manager so each stop-test begins with a running container.
// `distrobox create` does not start the container — enter does — so we start
// it directly to keep these tests focused on the stop command.
func createStopContainer(t *testing.T, h *Harness, name string) {
	t.Helper()
	h.MustDistrobox("create", "--yes", "--image", SmokeImage, "--name", name)
	if !h.ContainerRunning(name) {
		h.MustCM("start", name)
	}
}

// TestStop_StopsRunningContainer creates and starts a container, then
// verifies that `distrobox stop --yes` brings it down without removing it.
func TestStop_StopsRunningContainer(t *testing.T) {
	h := New(t)
	name := h.NewName("stops-running")
	createStopContainer(t, h, name)

	if !h.ContainerRunning(name) {
		t.Fatalf("expected container %s to be running before stop", name)
	}

	h.MustDistrobox("stop", "--yes", name)

	if h.ContainerRunning(name) {
		t.Fatalf("expected container %s to be stopped after `distrobox stop`", name)
	}
	if !h.ContainerExists(name) {
		t.Fatalf("expected container %s to still exist after stop (stop should not remove)", name)
	}
}

// TestStop_AllFlag confirms that `distrobox stop --yes --all` stops every
// distrobox-managed container in one call.
func TestStop_AllFlag(t *testing.T) {
	h := New(t)
	name1 := h.NewName("all-1")
	name2 := h.NewName("all-2")
	createStopContainer(t, h, name1)
	createStopContainer(t, h, name2)

	if !h.ContainerRunning(name1) || !h.ContainerRunning(name2) {
		t.Fatalf("expected both containers to be running before stop --all")
	}

	h.MustDistrobox("stop", "--yes", "--all")

	if h.ContainerRunning(name1) {
		t.Fatalf("expected container %s to be stopped by --all", name1)
	}
	if h.ContainerRunning(name2) {
		t.Fatalf("expected container %s to be stopped by --all", name2)
	}
}

// TestStop_NonExistent ensures that stopping a container that does not exist
// terminates (does not hang) — the exit code is allowed to be either 0 or
// non-zero, but the harness timeout would fire if the command blocked.
func TestStop_NonExistent(t *testing.T) {
	h := New(t)
	// Use NewName to keep the name unique and ensure it isn't a real container.
	name := h.NewName("nonexistent-xyz")

	// Do not assert on err: distrobox may treat missing containers as fatal or
	// as a no-op; the contract for this smoke test is that it returns at all.
	_, _ = h.Distrobox("stop", "--yes", name)
}

// TestStop_AlreadyStopped guarantees that stopping a container twice in a row
// does not crash distrobox (the second call should either be a no-op or fail
// cleanly).
func TestStop_AlreadyStopped(t *testing.T) {
	h := New(t)
	name := h.NewName("already-stopped")
	createStopContainer(t, h, name)

	h.MustDistrobox("stop", "--yes", name)
	if h.ContainerRunning(name) {
		t.Fatalf("expected container %s to be stopped after first stop", name)
	}

	// Second stop: contract is "does not crash". Distrobox is allowed to exit
	// non-zero here, but we should still get a clean return from the harness.
	_, _ = h.Distrobox("stop", "--yes", name)

	if h.ContainerRunning(name) {
		t.Fatalf("expected container %s to remain stopped after second stop", name)
	}
}
