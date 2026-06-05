#!/usr/bin/env bash
# Entry point used inside the docker-in-docker test host. Starts dockerd,
# waits for the socket, pre-pulls the smoke image, warms the image once so
# distrobox-init's apk-install phase is baked into the rootfs, then hands
# control over to the compiled go-test binary.
set -euo pipefail

# Launch dockerd via the upstream wrapper which configures cgroups, iptables
# and storage drivers for nested execution.
dockerd-entrypoint.sh dockerd >/var/log/dockerd.log 2>&1 &

for i in $(seq 1 60); do
	if docker info >/dev/null 2>&1; then
		break
	fi
	if [ "${i}" = "60" ]; then
		echo "dockerd did not become ready within 60s. Last log lines:" >&2
		tail -n 50 /var/log/dockerd.log >&2 || true
		exit 1
	fi
	sleep 1
done

IMAGE="${DBX_SMOKE_IMAGE:-docker.io/library/alpine:3.21}"
for attempt in 1 2 3 4 5; do
	if docker pull "${IMAGE}" >/dev/null 2>/tmp/pull.err; then
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
		docker stop --time 5 "${WARMUP_NAME}" >/dev/null 2>&1 &&
		docker commit "${WARMUP_NAME}" "${IMAGE}" >/dev/null &&
		docker rm --force "${WARMUP_NAME}" >/dev/null
} >"${WARMUP_LOG}" 2>&1; then
	echo "Warmup failed. Log:" >&2
	cat "${WARMUP_LOG}" >&2
	exit 1
fi

exec /work/distrobox-cli-tests -test.v "$@"
