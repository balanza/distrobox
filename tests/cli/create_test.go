//go:build cli

package cli

import (
	"strings"
	"testing"
)

// TestCreate_BasicImage exercises the happy path: ask distrobox to create a
// container from the alpine smoke image and verify the container exists with
// the distrobox manager label set. This is the smoke test every other create
// scenario builds on.
func TestCreate_BasicImage(t *testing.T) {
	h := New(t)
	name := h.NewName("basic")

	h.MustDistrobox(
		"create",
		"--yes",
		"--image", SmokeImage,
		"--name", name,
	)

	if !h.ContainerExists(name) {
		t.Fatalf("expected container %q to exist after create", name)
	}

	manager := h.Inspect(name, "{{.Config.Labels.manager}}")
	if manager != "distrobox" {
		t.Fatalf("expected Config.Labels.manager=distrobox, got %q", manager)
	}
}

// TestCreate_DryRunDoesNotCreate confirms that --dry-run prints the would-be
// command without touching the container manager. The container must not
// exist afterwards, and the output should mention the underlying create
// invocation.
func TestCreate_DryRunDoesNotCreate(t *testing.T) {
	h := New(t)
	name := h.NewName("dryrun")

	out := h.MustDistrobox(
		"create",
		"--yes",
		"--dry-run",
		"--image", SmokeImage,
		"--name", name,
	)

	if !strings.Contains(strings.ToLower(out), "create") {
		t.Fatalf("expected dry-run output to mention 'create', got:\n%s", out)
	}

	if h.ContainerExists(name) {
		t.Fatalf("dry-run created container %q; it should not exist", name)
	}
}

// TestCreate_WithHostname verifies the --hostname flag is forwarded to the
// container manager and persists in the container's Config.Hostname.
func TestCreate_WithHostname(t *testing.T) {
	h := New(t)
	name := h.NewName("hostname")

	h.MustDistrobox(
		"create",
		"--yes",
		"--image", SmokeImage,
		"--name", name,
		"--hostname", "foobar",
	)

	hostname := h.Inspect(name, "{{.Config.Hostname}}")
	if hostname != "foobar" {
		t.Fatalf("expected Config.Hostname=foobar, got %q", hostname)
	}
}

// TestCreate_RejectsDuplicate asserts that running `create` twice with the
// same name fails (non-zero exit) and reports the existing container. We do
// not assert on a particular wording but require either the name or an
// "exists" hint to surface.
func TestCreate_RejectsDuplicate(t *testing.T) {
	h := New(t)
	name := h.NewName("duplicate")

	h.MustDistrobox(
		"create",
		"--yes",
		"--image", SmokeImage,
		"--name", name,
	)

	out, err := h.Distrobox(
		"create",
		"--yes",
		"--image", SmokeImage,
		"--name", name,
	)
	if err == nil {
		t.Fatalf("expected second create to fail, got nil error\noutput:\n%s", out)
	}

	if !strings.Contains(strings.ToLower(out), "exist") {
		t.Fatalf("expected duplicate-create output to mention 'exist', got:\n%s", out)
	}
}

// TestCreate_WithAdditionalFlags exercises --additional-flags by injecting a
// custom env variable into the container and asserting the env survives all
// the way to the container manager spec.
func TestCreate_WithAdditionalFlags(t *testing.T) {
	h := New(t)
	name := h.NewName("addflags")

	h.MustDistrobox(
		"create",
		"--yes",
		"--image", SmokeImage,
		"--name", name,
		"--additional-flags", "--env MY_VAR=value",
	)

	env := h.Inspect(name, "{{.Config.Env}}")
	if !strings.Contains(env, "MY_VAR=value") {
		t.Fatalf("expected Config.Env to contain MY_VAR=value, got %q", env)
	}
}

// TestCreate_WithVolume verifies the --volume flag survives to the container
// manager: after create the container's Mounts list must include /host-tmp
// as a destination.
func TestCreate_WithVolume(t *testing.T) {
	h := New(t)
	name := h.NewName("volume")

	h.MustDistrobox(
		"create",
		"--yes",
		"--image", SmokeImage,
		"--name", name,
		"--volume", "/tmp:/host-tmp:ro",
	)

	mounts := h.Inspect(name, "{{range .Mounts}}{{.Destination}} {{end}}")
	if !strings.Contains(mounts, "/host-tmp") {
		t.Fatalf("expected mounts to contain /host-tmp, got %q", mounts)
	}
}
