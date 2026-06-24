SCENARIO_DESCRIPTION="raw 'podman run --rm' (baseline for 22-ephemeral-true)"
SCENARIO_WARMUP=1
SCENARIO_RUNS=5

RAW_RUN_IMAGE="docker.io/library/alpine:latest"

scenario_setup()   { :; }
scenario_command() { printf 'podman run --rm %s /bin/true\n' "$RAW_RUN_IMAGE"; }
scenario_cleanup() { :; }
