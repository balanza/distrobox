// Package cli holds end-to-end smoke tests that invoke the built distrobox
// binary as a CLI against a real container manager. The tests are compiled
// only when the "cli" build tag is set so they do not run with the regular
// `make test` target.
//
// The suite is meant to be executed inside an isolated container provisioned
// by tests/cli/run.sh, but each individual test only requires the
// distrobox binary on PATH (or via DISTROBOX_BIN) and a working container
// manager identified by DBX_CONTAINER_MANAGER.
package cli
