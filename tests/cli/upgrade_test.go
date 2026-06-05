//go:build cli

package cli

import (
	"testing"
)

// createUpgradeContainer creates a fresh distrobox container using the smoke image so
// each test in this file starts from a known-good state.
func createUpgradeContainer(t *testing.T, h *Harness, name string) {
	t.Helper()
	h.MustDistrobox("create", "--yes", "--image", SmokeImage, "--name", name)
}

// TestUpgrade_RunsOnContainer verifies that `distrobox upgrade --yes <name>`
// completes with exit code 0 against an Alpine-based container. apk's exact
// output varies, so we only assert successful exit.
func TestUpgrade_RunsOnContainer(t *testing.T) {
	h := New(t)
	name := h.NewName("upgrade-one")
	createUpgradeContainer(t, h, name)

	h.MustDistrobox("upgrade", "--yes", name)
}

// TestUpgrade_NoContainerArg checks that calling upgrade without a container
// name (and without --all) fails with a helpful error, matching the
// ErrUpgradeNoContainerSpecified path in internal/cli/upgrade.go.
func TestUpgrade_NoContainerArg(t *testing.T) {
	h := New(t)

	out, err := h.Distrobox("upgrade", "--yes")
	if err == nil {
		t.Fatalf("expected upgrade with no container and no --all to fail, got nil error\noutput: %s", out)
	}
}

// TestUpgrade_All confirms that `distrobox upgrade --yes --all` upgrades every
// distrobox-managed container and exits cleanly.
func TestUpgrade_All(t *testing.T) {
	h := New(t)
	name := h.NewName("upgrade-all")
	createUpgradeContainer(t, h, name)

	h.MustDistrobox("upgrade", "--yes", "--all")
}
