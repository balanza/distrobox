SCENARIO_DESCRIPTION="raw 'podman ps -a' (baseline for 10-list-empty: pure podman query cost, no wrapper)"
SCENARIO_WARMUP=3
SCENARIO_RUNS=30

scenario_setup()   { :; }
scenario_command() { printf 'podman ps -a\n'; }
scenario_cleanup() { :; }
