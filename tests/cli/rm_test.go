//go:build cli

package cli

import (
	"testing"
)

// TestRm_RemovesContainer creates a container then removes it with
// `distrobox rm --force` and asserts the container is no longer known to the
// underlying container manager.
func TestRm_RemovesContainer(t *testing.T) {
	h := New(t)
	name := h.NewName("rmbasic")

	h.MustDistrobox(
		"create",
		"--yes",
		"--image", SmokeImage,
		"--name", name,
	)
	if !h.ContainerExists(name) {
		t.Fatalf("setup failed: container %q does not exist after create", name)
	}

	h.MustDistrobox("rm", "--force", "--yes", name)

	if h.ContainerExists(name) {
		t.Fatalf("expected container %q to be gone after rm --force", name)
	}
}

// TestRm_ForceRunningContainer verifies that --force can remove a container
// even while it is in the running state. distrobox starts the container as
// part of create, so we explicitly assert running before removing.
func TestRm_ForceRunningContainer(t *testing.T) {
	h := New(t)
	name := h.NewName("rmrunning")

	h.MustDistrobox(
		"create",
		"--yes",
		"--image", SmokeImage,
		"--name", name,
	)

	// distrobox create leaves the container in a "Created" state until
	// enter starts it; some providers report Running=true immediately while
	// others do not. Make sure it is running before asserting --force can
	// reap a live container.
	if !h.ContainerRunning(name) {
		h.MustCM("start", name)
	}
	if !h.ContainerRunning(name) {
		t.Fatalf("expected container %q to be running before rm --force", name)
	}

	h.MustDistrobox("rm", "--force", "--yes", name)

	if h.ContainerExists(name) {
		t.Fatalf("expected container %q to be gone after rm --force on running container", name)
	}
}

// TestRm_NonExistentSilent asserts that asking distrobox to remove a name
// the container manager does not know about does not panic or hang. The
// command may exit zero or non-zero depending on the implementation; both
// outcomes are acceptable here, we only care that the binary terminates.
func TestRm_NonExistentSilent(t *testing.T) {
	h := New(t)
	name := h.NewName("nonexistent")

	// Make sure the chosen name truly does not exist.
	if h.ContainerExists(name) {
		t.Fatalf("precondition failed: container %q unexpectedly exists", name)
	}

	// We intentionally do not check the error: distrobox may or may not
	// surface a missing-container case as an error code. The contract this
	// test enforces is "binary completes without panic".
	_, _ = h.Distrobox("rm", "--force", "--yes", name)

	// And the container still does not exist afterwards.
	if h.ContainerExists(name) {
		t.Fatalf("container %q should still not exist after rm of nonexistent name", name)
	}
}

// TestRm_AllFlag creates two containers and asserts that
// `distrobox rm --all --force --yes` removes both of them in one shot.
func TestRm_AllFlag(t *testing.T) {
	h := New(t)
	first := h.NewName("rmall1")
	second := h.NewName("rmall2")

	h.MustDistrobox(
		"create",
		"--yes",
		"--image", SmokeImage,
		"--name", first,
	)
	h.MustDistrobox(
		"create",
		"--yes",
		"--image", SmokeImage,
		"--name", second,
	)
	if !h.ContainerExists(first) || !h.ContainerExists(second) {
		t.Fatalf("setup failed: both containers should exist, got first=%v second=%v",
			h.ContainerExists(first), h.ContainerExists(second))
	}

	h.MustDistrobox("rm", "--all", "--force", "--yes")

	if h.ContainerExists(first) {
		t.Fatalf("expected container %q to be gone after rm --all --force", first)
	}
	if h.ContainerExists(second) {
		t.Fatalf("expected container %q to be gone after rm --all --force", second)
	}
}
