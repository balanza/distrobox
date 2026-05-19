#!/usr/bin/env bash
# tests/cli/run.sh — orchestrate the CLI smoke-test suite inside an isolated
# container that already has the target container manager installed.
#
# Inputs:
#   $1 (or $CONTAINER_MANAGER) — podman | docker
# Environment:
#   TEST_ENGINE  — the container manager used to run the OUTER test host.
#                  Defaults to whichever of podman/docker is available, with
#                  podman preferred.
#   TEST_ARGS    — extra arguments forwarded to `go test` inside the host
#                  (e.g. -test.run TestCreate_).
#   KEEP_HOST    — when "1", the test host is started detached and left
#                  running for post-mortem inspection.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

CM="${1:-${CONTAINER_MANAGER:-}}"
case "${CM}" in
	podman | docker) ;;
	"")
		echo "usage: $0 <podman|docker>" >&2
		exit 64
		;;
	*)
		echo "unknown container manager: ${CM}" >&2
		exit 64
		;;
esac

TEST_ENGINE="${TEST_ENGINE:-}"
if [ -z "${TEST_ENGINE}" ]; then
	if command -v podman >/dev/null 2>&1; then
		TEST_ENGINE=podman
	elif command -v docker >/dev/null 2>&1; then
		TEST_ENGINE=docker
	else
		echo "no outer container engine (podman or docker) available" >&2
		exit 1
	fi
fi

KEEP_HOST="${KEEP_HOST:-0}"
TEST_ARGS="${TEST_ARGS:-}"

IMAGE_TAG="distrobox-cli-tests:${CM}"
CONTAINERFILE="${SCRIPT_DIR}/Containerfile.${CM}"

echo "==> Building distrobox binary"
(cd "${REPO_DIR}" && make build)

echo "==> Compiling test binary"
(
	cd "${REPO_DIR}"
	CGO_ENABLED=0 GOOS=linux go test -tags=cli -c \
		-o "${REPO_DIR}/bin/distrobox-cli-tests" \
		./tests/cli
)

echo "==> Building test host image (${TEST_ENGINE} build -t ${IMAGE_TAG})"
BUILD_CTX="$(mktemp -d)"
trap 'rm -rf "${BUILD_CTX}"' EXIT
cp "${REPO_DIR}/bin/distrobox" "${BUILD_CTX}/distrobox"
cp "${REPO_DIR}/bin/distrobox-cli-tests" "${BUILD_CTX}/distrobox-cli-tests"
cp "${SCRIPT_DIR}/entrypoint-podman.sh" "${BUILD_CTX}/entrypoint-podman.sh"
cp "${SCRIPT_DIR}/entrypoint-docker.sh" "${BUILD_CTX}/entrypoint-docker.sh"
chmod +x "${BUILD_CTX}/entrypoint-podman.sh" "${BUILD_CTX}/entrypoint-docker.sh"

# Assemble a tiny context wrapping the source Containerfile but with extra
# COPY steps so the binaries and entrypoints land in the right place
# regardless of where the user invoked `make` from.
cat > "${BUILD_CTX}/Containerfile" <<EOF
$(cat "${CONTAINERFILE}")

COPY --chown=root:root distrobox /usr/local/bin/distrobox
COPY --chown=root:root distrobox-cli-tests /work/distrobox-cli-tests
COPY --chown=root:root entrypoint-podman.sh /work/entrypoint-podman.sh
COPY --chown=root:root entrypoint-docker.sh /work/entrypoint-docker.sh
EOF

# Retry the image build a few times — pulling base images from public
# registries occasionally fails with transient TLS errors that resolve on the
# next attempt.
build_image()
{
	local attempt
	for attempt in 1 2 3; do
		if "${TEST_ENGINE}" build -t "${IMAGE_TAG}" -f "${BUILD_CTX}/Containerfile" "${BUILD_CTX}"; then
			return 0
		fi
		if [ "${attempt}" = "3" ]; then
			return 1
		fi
		echo "==> Build attempt ${attempt} failed, retrying in $((attempt * 5))s" >&2
		sleep "$((attempt * 5))"
	done
}
build_image

# Container-manager-specific runtime flags. Podman-in-podman runs rootless
# and only needs fuse + cgroups; docker-in-docker needs the full --privileged
# treatment because dockerd manipulates iptables and cgroups directly.
RUN_FLAGS=(--rm)
case "${CM}" in
	podman)
		RUN_FLAGS+=(
			--privileged
			--security-opt label=disable
			--device /dev/fuse
		)
		;;
	docker)
		RUN_FLAGS+=(
			--privileged
		)
		;;
esac

if [ -n "${TEST_ARGS}" ]; then
	# shellcheck disable=SC2086
	set -- ${TEST_ARGS}
else
	set --
fi

if [ "${KEEP_HOST}" = "1" ]; then
	echo "==> Starting test host detached (KEEP_HOST=1)"
	"${TEST_ENGINE}" run -d "${RUN_FLAGS[@]}" --name "distrobox-cli-tests-${CM}" "${IMAGE_TAG}" "$@"
	echo "Attach with: ${TEST_ENGINE} logs -f distrobox-cli-tests-${CM}"
	exit 0
fi

echo "==> Running CLI tests inside ${IMAGE_TAG}"
"${TEST_ENGINE}" run "${RUN_FLAGS[@]}" "${IMAGE_TAG}" "$@"
