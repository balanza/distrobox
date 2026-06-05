//go:build cli

package cli

import (
	"strings"
	"testing"
)

// TestSanity_VersionPrints validates that the harness can invoke the binary
// at all. It must pass before any other test is meaningful.
func TestSanity_VersionPrints(t *testing.T) {
	h := New(t)
	out := h.MustDistrobox("--version")
	if !strings.Contains(out, "distrobox") {
		t.Fatalf("expected version output to mention 'distrobox', got: %q", out)
	}
}

// TestSanity_HelpExits ensures the help flag does not error and lists every
// top-level command so we notice if a command is dropped from the binary.
func TestSanity_HelpExits(t *testing.T) {
	h := New(t)
	out := h.MustDistrobox("--help")
	for _, want := range []string{
		"assemble",
		"create",
		"enter",
		"ephemeral",
		"generate-entry",
		"list",
		"rm",
		"stop",
		"upgrade",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("help output missing %q\n%s", want, out)
		}
	}
}

// TestSanity_ContainerManagerAvailable proves the test host actually has the
// container manager that the suite was asked to exercise. If this fails the
// outer setup is broken (Containerfile, run.sh) and there is no point
// continuing to per-command tests.
func TestSanity_ContainerManagerAvailable(t *testing.T) {
	h := New(t)
	out, err := h.CM("version", "--format", "{{.Server.Version}}")
	if err != nil {
		t.Fatalf("%s version failed: %v\n%s", h.ContainerManager, err, out)
	}
	if strings.TrimSpace(out) == "" {
		t.Fatalf("%s version returned empty string", h.ContainerManager)
	}
}
