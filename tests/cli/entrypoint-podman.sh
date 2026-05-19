#!/usr/bin/env bash
# Entry point used inside the podman-in-podman test host. Pre-pulls the
# smoke image (with retries so a single registry blip doesn't fail an entire
# test run) and then hands control over to the compiled go-test binary.
set -euo pipefail

IMAGE="${DBX_SMOKE_IMAGE:-docker.io/library/alpine:3.21}"

for attempt in 1 2 3 4 5; do
	if podman pull "${IMAGE}" >/dev/null 2>/tmp/pull.err; then
		break
	fi
	if [ "${attempt}" = "5" ]; then
		echo "Failed to pull ${IMAGE} after ${attempt} attempts:" >&2
		cat /tmp/pull.err >&2
		exit 1
	fi
	sleep "${attempt}"
done

exec /work/distrobox-cli-tests -test.v "$@"
