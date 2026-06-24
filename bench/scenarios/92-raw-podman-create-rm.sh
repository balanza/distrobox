SCENARIO_DESCRIPTION="raw 'podman create' + 'podman rm -f' (baseline for 20-create-rm)"
SCENARIO_WARMUP=1
SCENARIO_RUNS=10

RAW_CREATE_RM_IMAGE="docker.io/library/alpine:latest"

scenario_setup() { :; }

scenario_command() {
    # Same uniqueness trick as 20-create-rm: defer $$ and $RANDOM expansion to runtime sh,
    # append tracker line BEFORE create so EXIT-trap cleanup catches mid-iteration crashes.
    printf "sh -c 'NAME=dbx-bench-%s-raw-create-rm-\$\$-\$RANDOM; printf \"%%s\\n\" \"\$NAME\" >> %s/containers.list; podman create --label distrobox.bench.run=%s --name \"\$NAME\" %s /bin/true >/dev/null && podman rm -f \"\$NAME\" >/dev/null'\n" \
        "$RUN_ID" "$RESULTS_DIR" "$RUN_ID" "$RAW_CREATE_RM_IMAGE"
}

scenario_cleanup() { :; }
