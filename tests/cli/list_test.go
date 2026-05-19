//go:build cli

package cli

import (
	"strings"
	"testing"
)

// TestList_EmptyState runs `distrobox list` on a fresh harness and asserts
// that no row for this test's container name is present. Other tests may be
// running in parallel inside the same process, so we cannot assume zero rows
// overall — only the absence of our own name.
func TestList_EmptyState(t *testing.T) {
	h := New(t)
	name := h.NewName("emptystate")

	out := h.MustDistrobox("list")
	if !strings.Contains(out, "NAME") {
		t.Fatalf("expected list output to include header column 'NAME', got:\n%s", out)
	}
	if strings.Contains(out, name) {
		t.Fatalf("expected list output to NOT contain %q (we never created it), got:\n%s", name, out)
	}
}

// TestList_ShowsCreated creates a container and asserts both the name and
// the image appear in `distrobox list` output. We only check the trailing
// path segment of the image so the assertion works regardless of how the
// container manager renders fully-qualified registry references.
func TestList_ShowsCreated(t *testing.T) {
	h := New(t)
	name := h.NewName("showscreated")

	h.MustDistrobox(
		"create",
		"--yes",
		"--image", SmokeImage,
		"--name", name,
	)

	out := h.MustDistrobox("list")
	if !strings.Contains(out, name) {
		t.Fatalf("expected list output to contain %q, got:\n%s", name, out)
	}
	if !strings.Contains(out, "alpine") {
		t.Fatalf("expected list output to mention the image (alpine), got:\n%s", out)
	}
}

// TestList_NoColorFlag asserts the --no-color flag is accepted and produces
// usable output that still contains the created container's name. We don't
// check the absence of ANSI escapes specifically — the harness pipes stdout
// to a buffer, so the binary already considers itself not on a TTY.
func TestList_NoColorFlag(t *testing.T) {
	h := New(t)
	name := h.NewName("nocolor")

	h.MustDistrobox(
		"create",
		"--yes",
		"--image", SmokeImage,
		"--name", name,
	)

	out := h.MustDistrobox("list", "--no-color")
	if !strings.Contains(out, name) {
		t.Fatalf("expected --no-color list output to contain %q, got:\n%s", name, out)
	}
}

// TestList_AliasLs verifies that `distrobox ls` aliases `distrobox list` and
// surfaces the same created container. We only assert on the presence of the
// name; column formatting is covered by TestList_ShowsCreated.
func TestList_AliasLs(t *testing.T) {
	h := New(t)
	name := h.NewName("lsalias")

	h.MustDistrobox(
		"create",
		"--yes",
		"--image", SmokeImage,
		"--name", name,
	)

	out := h.MustDistrobox("ls")
	if !strings.Contains(out, name) {
		t.Fatalf("expected `ls` alias output to contain %q, got:\n%s", name, out)
	}
}
