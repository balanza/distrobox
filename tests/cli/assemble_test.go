//go:build cli

package cli

import (
	"fmt"
	"strings"
	"testing"
)

// TestAssemble_CreateFromFile creates a single container from a manifest file
// and verifies the container exists and uses the smoke image.
func TestAssemble_CreateFromFile(t *testing.T) {
	h := New(t)
	name := h.NewName("solo")
	manifest := fmt.Sprintf("[%s]\nimage=%s\n", name, SmokeImage)
	path := h.WriteFile("manifest.ini", manifest)

	out := h.MustDistrobox("assemble", "create", "--file", path)
	if !h.ContainerExists(name) {
		t.Fatalf("expected container %q to exist after assemble create\noutput:\n%s", name, out)
	}

	image := h.Inspect(name, "{{.Config.Image}}")
	if !strings.Contains(image, "alpine") {
		t.Fatalf("expected container image to mention 'alpine', got %q", image)
	}
}

// TestAssemble_CreateMultiple verifies that a manifest with two entries
// produces both containers.
func TestAssemble_CreateMultiple(t *testing.T) {
	h := New(t)
	first := h.NewName("first")
	second := h.NewName("second")
	manifest := fmt.Sprintf(
		"[%s]\nimage=%s\n\n[%s]\nimage=%s\n",
		first, SmokeImage, second, SmokeImage,
	)
	path := h.WriteFile("manifest.ini", manifest)

	out := h.MustDistrobox("assemble", "create", "--file", path)
	if !h.ContainerExists(first) {
		t.Fatalf("expected container %q to exist after assemble create\noutput:\n%s", first, out)
	}
	if !h.ContainerExists(second) {
		t.Fatalf("expected container %q to exist after assemble create\noutput:\n%s", second, out)
	}
}

// TestAssemble_NameFilter only creates the named entry from a manifest with
// multiple entries.
func TestAssemble_NameFilter(t *testing.T) {
	h := New(t)
	wanted := h.NewName("wanted")
	skipped := h.NewName("skipped")
	manifest := fmt.Sprintf(
		"[%s]\nimage=%s\n\n[%s]\nimage=%s\n",
		wanted, SmokeImage, skipped, SmokeImage,
	)
	path := h.WriteFile("manifest.ini", manifest)

	out := h.MustDistrobox("assemble", "create", "--file", path, "--name", wanted)
	if !h.ContainerExists(wanted) {
		t.Fatalf("expected container %q to exist when filtered by --name\noutput:\n%s", wanted, out)
	}
	if h.ContainerExists(skipped) {
		t.Fatalf("expected container %q to NOT exist when filtered by --name\noutput:\n%s", skipped, out)
	}
}

// TestAssemble_DryRun runs assemble with --dry-run and asserts that no
// container is actually created on the host.
func TestAssemble_DryRun(t *testing.T) {
	h := New(t)
	name := h.NewName("dry")
	manifest := fmt.Sprintf("[%s]\nimage=%s\n", name, SmokeImage)
	path := h.WriteFile("manifest.ini", manifest)

	out := h.MustDistrobox("assemble", "create", "--file", path, "--dry-run")
	if h.ContainerExists(name) {
		t.Fatalf("expected container %q to NOT exist after --dry-run\noutput:\n%s", name, out)
	}
}

// TestAssemble_RmFromFile creates containers via assemble, then deletes them
// with assemble rm and asserts they are gone.
func TestAssemble_RmFromFile(t *testing.T) {
	h := New(t)
	name := h.NewName("rm")
	manifest := fmt.Sprintf("[%s]\nimage=%s\n", name, SmokeImage)
	path := h.WriteFile("manifest.ini", manifest)

	h.MustDistrobox("assemble", "create", "--file", path)
	if !h.ContainerExists(name) {
		t.Fatalf("precondition: container %q should exist before rm", name)
	}

	out := h.MustDistrobox("assemble", "rm", "--file", path)
	if h.ContainerExists(name) {
		t.Fatalf("expected container %q to be removed after assemble rm\noutput:\n%s", name, out)
	}
}

// TestAssemble_Replace verifies that re-running assemble with --replace
// keeps the container present (replaces, not duplicates or fails) and that
// the resulting container is a fresh one (different container ID).
func TestAssemble_Replace(t *testing.T) {
	h := New(t)
	name := h.NewName("replace")
	manifest := fmt.Sprintf("[%s]\nimage=%s\n", name, SmokeImage)
	path := h.WriteFile("manifest.ini", manifest)

	h.MustDistrobox("assemble", "create", "--file", path)
	if !h.ContainerExists(name) {
		t.Fatalf("precondition: container %q should exist before replace", name)
	}
	idBefore := h.Inspect(name, "{{.Id}}")

	out := h.MustDistrobox("assemble", "create", "--file", path, "--replace")
	if !h.ContainerExists(name) {
		t.Fatalf("expected container %q to exist after --replace\noutput:\n%s", name, out)
	}
	idAfter := h.Inspect(name, "{{.Id}}")
	if idBefore == idAfter {
		t.Fatalf("expected container %q to be a fresh container after --replace; id unchanged: %s",
			name, idBefore)
	}
}
