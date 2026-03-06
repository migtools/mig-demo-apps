#!/usr/bin/env bash
# OpenShift smoke test: wait for pod + route, then run CRUD assertions via curl.
# Usage: ./test/openshift-smoke.sh <namespace> <route_base_url>
# Example: ./test/openshift-smoke.sh mongo-persistent http://todolist-route-mongo-persistent.apps.example.com

set -e

NAMESPACE="${1:?Usage: $0 <namespace> <route_base_url>}"
BASE_URL="${2:?Usage: $0 <namespace> <route_base_url>}"
# Trim trailing slash
BASE_URL="${BASE_URL%/}"
POD_TIMEOUT=300
READYZ_TIMEOUT=120
READYZ_INTERVAL=5

echo "OpenShift smoke test: namespace=$NAMESPACE url=$BASE_URL"

# 1. Poll until pod is 1/1 Running (up to 5 min)
echo "Waiting for pod in $NAMESPACE to be 1/1 Running (up to ${POD_TIMEOUT}s)..."
start=$(date +%s)
while true; do
  if oc get pods -n "$NAMESPACE" -l app=todolist --no-headers 2>/dev/null | grep -q '1/1.*Running'; then
    echo "Pod is Running."
    break
  fi
  now=$(date +%s)
  if (( now - start >= POD_TIMEOUT )); then
    echo "ERROR: Pod did not become 1/1 Running within ${POD_TIMEOUT}s" >&2
    oc get pods -n "$NAMESPACE" >&2
    exit 1
  fi
  sleep 5
done

# 2. Poll GET /readyz until 200 (proves DB started and app connected)
echo "Waiting for /readyz to return 200 (up to ${READYZ_TIMEOUT}s)..."
start=$(date +%s)
while true; do
  code=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 5 "$BASE_URL/readyz" 2>/dev/null || echo "000")
  if [[ "$code" == "200" ]]; then
    echo "readyz returned 200 (DB connected)."
    break
  fi
  now=$(date +%s)
  if (( now - start >= READYZ_TIMEOUT )); then
    echo "ERROR: /readyz did not return 200 within ${READYZ_TIMEOUT}s (last code: $code)" >&2
    exit 1
  fi
  sleep "$READYZ_INTERVAL"
done

# 3. CRUD sequence via curl
echo "Running CRUD smoke checks..."

# GET /healthz -> 200, body has "alive"
code=$(curl -s -o /tmp/smoke_body.txt -w "%{http_code}" "$BASE_URL/healthz")
[[ "$code" == "200" ]] || { echo "FAIL: GET /healthz expected 200, got $code"; exit 1; }
grep -q '"alive".*true' /tmp/smoke_body.txt || { echo "FAIL: /healthz body missing alive:true"; exit 1; }
echo "  GET /healthz OK"

# POST /todo -> 200, parse Id from JSON array (result[0].Id)
create_resp=$(curl -s -w "\n%{http_code}" -X POST -d "description=smoke-test" -H "Content-Type: application/x-www-form-urlencoded" "$BASE_URL/todo")
create_body=$(echo "$create_resp" | head -n -1)
code=$(echo "$create_resp" | tail -n 1)
[[ "$code" == "200" ]] || { echo "FAIL: POST /todo expected 200, got $code"; echo "$create_body" | head -c 200; exit 1; }
# Extract Id (first "Id":"<value>" in the array); portable sed
id=$(echo "$create_body" | sed -n 's/.*"Id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)
[[ -n "$id" ]] || { echo "FAIL: POST /todo response missing Id"; echo "$create_body" | head -c 200; exit 1; }
echo "  POST /todo OK (id=$id)"

# GET /todo-incomplete -> 200, body contains our id
code=$(curl -s -o /tmp/smoke_body.txt -w "%{http_code}" "$BASE_URL/todo-incomplete")
[[ "$code" == "200" ]] || { echo "FAIL: GET /todo-incomplete expected 200, got $code"; exit 1; }
grep -q "\"Id\":\"$id\"" /tmp/smoke_body.txt || grep -q "\"Id\": \"$id\"" /tmp/smoke_body.txt || { echo "FAIL: todo-incomplete missing created item"; exit 1; }
echo "  GET /todo-incomplete OK"

# POST /todo/{id} completed=true -> 200
code=$(curl -s -o /tmp/smoke_body.txt -w "%{http_code}" -X POST -d "completed=true" -H "Content-Type: application/x-www-form-urlencoded" "$BASE_URL/todo/$id")
[[ "$code" == "200" ]] || { echo "FAIL: POST /todo/$id expected 200, got $code"; exit 1; }
grep -q '"updated"' /tmp/smoke_body.txt || { echo "FAIL: update response missing updated"; exit 1; }
echo "  POST /todo/$id (complete) OK"

# GET /todo-completed -> 200, body contains our id
code=$(curl -s -o /tmp/smoke_body.txt -w "%{http_code}" "$BASE_URL/todo-completed")
[[ "$code" == "200" ]] || { echo "FAIL: GET /todo-completed expected 200, got $code"; exit 1; }
grep -q "\"Id\":\"$id\"" /tmp/smoke_body.txt || grep -q "\"Id\": \"$id\"" /tmp/smoke_body.txt || { echo "FAIL: todo-completed missing item"; exit 1; }
echo "  GET /todo-completed OK"

# DELETE /todo/{id} -> 200
code=$(curl -s -o /tmp/smoke_body.txt -w "%{http_code}" -X DELETE "$BASE_URL/todo/$id")
[[ "$code" == "200" ]] || { echo "FAIL: DELETE /todo/$id expected 200, got $code"; exit 1; }
grep -q '"deleted"' /tmp/smoke_body.txt || { echo "FAIL: delete response missing deleted"; exit 1; }
echo "  DELETE /todo/$id OK"

# GET /todo/{id} -> 404 (delete persisted to DB)
code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/todo/$id")
[[ "$code" == "404" ]] || { echo "FAIL: GET /todo/$id after delete expected 404, got $code"; exit 1; }
echo "  GET /todo/$id after delete -> 404 OK"

rm -f /tmp/smoke_body.txt
echo "All smoke checks passed."
