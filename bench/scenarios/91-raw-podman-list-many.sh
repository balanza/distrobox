SCENARIO_DESCRIPTION="raw 'podman ps -a' with 20 dummy containers (baseline for 11-list-many)"
SCENARIO_WARMUP=3
SCENARIO_RUNS=30

LIST_MANY_COUNT=20
LIST_MANY_IMAGE="docker.io/library/alpine:latest"

scenario_setup() {
    local i name
    for i in $(seq 1 "$LIST_MANY_COUNT"); do
        name="dbx-bench-${RUN_ID}-raw-list-many-$(printf '%02d' "$i")"
        container_create_podman_direct "$name" "$LIST_MANY_IMAGE"
    done
}

scenario_command() { printf 'podman ps -a\n'; }
scenario_cleanup() { :; }
