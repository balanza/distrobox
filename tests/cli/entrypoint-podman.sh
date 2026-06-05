#!/usr/bin/env bash
# Entry point used inside the podman-in-podman test host. Pre-pulls the
# smoke image (with retries so a single registry blip doesn't fail an entire
# test run), warms the image once so distrobox-init's apk-install phase is
# baked into the rootfs, then hands control over to the compiled go-test
# binary.
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

# distrobox-init re-runs the full apk-install on every fresh container
# unless /.containersetupdone is present. Pay that cost once here, then
# retag the warmed-up rootfs back over the smoke image — per-test
# containers inherit the marker and short-circuit the multi-minute install
# on first enter, keeping individual tests under the harness 5-minute
# command timeout.
echo "==> Warming smoke image ${IMAGE}"
WARMUP_NAME="distrobox-warmup"
WARMUP_LOG="/tmp/warmup.log"
if ! {
	distrobox create --yes --image "${IMAGE}" --name "${WARMUP_NAME}" &&
		distrobox enter --no-tty "${WARMUP_NAME}" -- true &&
		podman stop --time 5 "${WARMUP_NAME}" >/dev/null 2>&1 &&
		podman commit "${WARMUP_NAME}" "${IMAGE}" >/dev/null &&
		podman rm --force "${WARMUP_NAME}" >/dev/null
} >"${WARMUP_LOG}" 2>&1; then
	echo "Warmup failed. Log:" >&2
	cat "${WARMUP_LOG}" >&2
	exit 1
fi

exec /work/distrobox-cli-tests -test.v "$@"
