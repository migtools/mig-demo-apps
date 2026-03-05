#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MANIFEST_DIR="$SCRIPT_DIR/../OPENSHIFT/mysql-persistent"
NAMESPACE="mysql-persistent"
ROUTE_NAME="todolist-route"
POD_TIMEOUT="180s"
NS_TEARDOWN_TIMEOUT=120

PASS=0
FAIL=0
RESULTS=()

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log()  { echo -e "${YELLOW}>>> $*${NC}"; }
pass() { echo -e "${GREEN}PASS: $*${NC}"; PASS=$((PASS+1)); RESULTS+=("PASS: $*"); }
fail() { echo -e "${RED}FAIL: $*${NC}"; FAIL=$((FAIL+1)); RESULTS+=("FAIL: $*"); }

wait_ns_gone() {
    local elapsed=0
    while oc get namespace "$NAMESPACE" 2>/dev/null | grep -q Terminating; do
        sleep 3
        elapsed=$((elapsed+3))
        if [ "$elapsed" -ge "$NS_TEARDOWN_TIMEOUT" ]; then
            echo "ERROR: namespace $NAMESPACE still terminating after ${NS_TEARDOWN_TIMEOUT}s" >&2
            return 1
        fi
    done
}

teardown() {
    local manifest="$1"
    local pvc="$2"
    log "Tearing down $manifest"
    oc delete -f "$manifest" --ignore-not-found 2>/dev/null || true
    oc delete -f "$pvc" --ignore-not-found 2>/dev/null || true
    wait_ns_gone
}

add_data() {
    local route="$1"
    local prefix="$2"
    for i in 1 2 3; do
        curl -sf -X POST "http://$route/todo" -d "description=${prefix}-${i}" >/dev/null
    done
}

check_data() {
    local route="$1"
    local prefix="$2"
    local items
    items=$(curl -sf "http://$route/todo-incomplete")
    local count
    count=$(echo "$items" | python3 -c "import sys,json; print(len([i for i in json.load(sys.stdin) if i['Description'].startswith('$prefix')]))" 2>/dev/null || echo 0)
    echo "$count"
}

run_test() {
    local test_name="$1"
    local manifest="$2"
    local pvc="$3"
    local timeout="$4"

    log "===== $test_name ====="

    # Ensure clean state
    if oc get namespace "$NAMESPACE" 2>/dev/null | grep -q "$NAMESPACE"; then
        teardown "$manifest" "$pvc"
    fi

    log "Deploying $manifest"
    oc apply -f "$manifest"
    oc apply -f "$pvc"

    log "Waiting for pod to be Ready (timeout: $timeout)"
    if ! oc wait --for=condition=Ready pod -l app=todolist -n "$NAMESPACE" --timeout="$timeout" 2>&1; then
        fail "$test_name - pod never became ready"
        oc logs -l app=todolist -n "$NAMESPACE" --tail=30 2>/dev/null || true
        teardown "$manifest" "$pvc"
        return
    fi

    local route
    route=$(oc get route "$ROUTE_NAME" -n "$NAMESPACE" -o jsonpath='{.spec.host}')

    local prefix="test-${test_name// /-}"
    log "Adding 3 test items (prefix: $prefix)"
    if ! add_data "$route" "$prefix"; then
        fail "$test_name - could not add data"
        teardown "$manifest" "$pvc"
        return
    fi

    local count_before
    count_before=$(check_data "$route" "$prefix")
    if [ "$count_before" -ne 3 ]; then
        fail "$test_name - expected 3 items before restart, got $count_before"
        teardown "$manifest" "$pvc"
        return
    fi
    log "Verified $count_before items present before restart"

    log "Deleting pod to trigger restart"
    oc delete pod -l app=todolist -n "$NAMESPACE"
    if ! oc wait --for=condition=Ready pod -l app=todolist -n "$NAMESPACE" --timeout="$timeout" 2>&1; then
        fail "$test_name - replacement pod never became ready"
        oc logs -l app=todolist -n "$NAMESPACE" --tail=30 2>/dev/null || true
        teardown "$manifest" "$pvc"
        return
    fi

    # Brief pause for the app to reconnect to the DB
    sleep 3

    local count_after
    count_after=$(check_data "$route" "$prefix")
    if [ "$count_after" -eq 3 ]; then
        pass "$test_name - all 3 items persisted after pod restart"
    else
        fail "$test_name - expected 3 items after restart, got $count_after"
    fi

    teardown "$manifest" "$pvc"
}

# --- Main ---
log "Starting mysql image persistence tests"
echo ""

run_test "mysql-persistent" \
    "$MANIFEST_DIR/mysql-persistent.yaml" \
    "$MANIFEST_DIR/pvc/default_sc.yaml" \
    "$POD_TIMEOUT"

run_test "mysql-persistent-csi" \
    "$MANIFEST_DIR/mysql-persistent-csi.yaml" \
    "$MANIFEST_DIR/pvc/default_sc.yaml" \
    "$POD_TIMEOUT"

run_test "mysql-persistent-twovol-csi" \
    "$MANIFEST_DIR/mysql-persistent-twovol-csi.yaml" \
    "$MANIFEST_DIR/pvc-twoVol/default_sc.yaml" \
    "$POD_TIMEOUT"

echo ""
log "===== Results ====="
for r in "${RESULTS[@]}"; do
    echo -e "  $r"
done
echo ""
echo -e "Total: ${GREEN}${PASS} passed${NC}, ${RED}${FAIL} failed${NC}"
exit "$FAIL"
