#!/usr/bin/env bash
# Implements DESIGN-010 RouteHandler local development process lifecycle.

start_dev_process() {
    local working_directory="$1"
    shift

    (
        cd "$working_directory"
        exec setsid "$@"
    ) &
    DEV_PROCESS_PID=$!
}

stop_dev_process() {
    local process_group_id="$1"

    [[ -z "$process_group_id" ]] && return 0
    kill -TERM -- "-$process_group_id" 2>/dev/null || true
}
