# todo2-go Design Document

## Purpose

`todo2-go` is a Velero test application for OpenShift. It provides a simple todo list web UI backed by either MariaDB or MongoDB in a single container. The primary requirements driving all design decisions are:

1. Velero backup/restore testing — the application must survive a PVC snapshot and restore with data intact.
2. OpenShift non-root SCC compatibility — the container must run correctly under any UID OpenShift injects.
3. Database flexibility — the same Go codebase supports both MariaDB and MongoDB, selected at runtime.
4. Resilient startup — the application tolerates slow or delayed database availability without crashing.

---

## Architecture Overview

```
┌──────────────────────────────────────────────────────┐
│                   Container (single pod)             │
│                                                      │
│  entrypoint.sh                                       │
│       │                                              │
│       ├── [if local DB] start MariaDB or MongoDB     │
│       │       └── wait for readiness                 │
│       └── exec Go HTTP server (:8000)                │
│                    │                                 │
│            ┌───────┴────────┐                        │
│            │  Gorilla Mux   │                        │
│            │  HTTP Router   │                        │
│            └───────┬────────┘                        │
│                    │                                 │
│            ┌───────┴────────┐                        │
│            │  api.Handler   │                        │
│            │  (handlers.go) │                        │
│            └───────┬────────┘                        │
│                    │  TodoStore interface             │
│         ┌──────────┴──────────┐                      │
│         │                     │                      │
│  mariadb.Store          mongodb.Store                │
│  (GORM + MySQL)         (mongo-driver)               │
│         │                     │                      │
│    TCP 3306             TCP 27017                    │
│         │                     │                      │
│    MariaDB              MongoDB                      │
│  /var/lib/mysql/data  /var/lib/mongodb               │
└──────────────────────────────────────────────────────┘
```

The application and database always share the same container. There is no sidecar pattern. External databases are supported (by setting `MYSQL_HOST` or `MONGO_URI` to a non-local host) but are not the primary use case.

---

## Go Module and Dependencies

Module: `github.com/weshayutin/todo2-go`
Language: Go 1.25

| Dependency                      | Purpose                                  |
|---------------------------------|------------------------------------------|
| `gorm.io/gorm`                  | ORM for MariaDB                          |
| `gorm.io/driver/mysql`          | GORM MySQL/MariaDB driver                |
| `go.mongodb.org/mongo-driver`   | Official MongoDB Go driver               |
| `github.com/gorilla/mux`        | HTTP router                              |
| `github.com/rs/cors`            | CORS middleware                          |
| `github.com/sirupsen/logrus`    | Structured logging                       |
| `github.com/stretchr/testify`   | Test assertions                          |

---

## Data Model

```go
// internal/model/todo.go
type TodoItem struct {
    ID          string `json:"id"`
    Description string `json:"description"`
    Completed   bool   `json:"completed"`
}
```

`ID` is always a `string`. MariaDB stores an auto-increment integer that is converted to string on read. MongoDB stores a `primitive.ObjectID` that is rendered as its hex string. This abstraction keeps the API layer database-agnostic.

---

## Store Interface

```go
// internal/store/store.go
type TodoStore interface {
    Create(ctx context.Context, description string) (*model.TodoItem, error)
    GetByCompleted(ctx context.Context, completed bool) ([]*model.TodoItem, error)
    GetByID(ctx context.Context, id string) (*model.TodoItem, error)
    Update(ctx context.Context, id string, completed bool) error
    Delete(ctx context.Context, id string) error
    Ping(ctx context.Context) error
    Close() error
}

var (
    ErrNotReady = errors.New("database not ready")
    ErrNotFound = errors.New("not found")
)
```

`ErrNotReady` is returned by any method before the background connection goroutine signals success. The HTTP handler layer maps this to HTTP 503. `ErrNotFound` maps to HTTP 404.

---

## Connection Resilience

Both store implementations connect to the database in a background goroutine using exponential backoff with jitter. The main goroutine (and HTTP server) start immediately; all DB-dependent endpoints return 503 until the background goroutine signals readiness via an `onReady` callback.

```
wait = 1s
loop:
  try connect
  if success:
    onReady()
    return
  wait = min(wait * 2, 30s)
  jitter = wait * rand(-0.1, 0.1)
  sleep(wait + jitter)
```

This means:
- The HTTP server (and liveness probe `/healthz`) is available immediately.
- `/readyz` returns 503 until `store.Ping()` succeeds (DB is actually reachable).
- OpenShift's `startupProbe` on `/healthz` allows the container to be marked live while waiting for DB init, which can take over a minute on first boot.

---

## HTTP Layer

`api.Handler` depends only on the `TodoStore` interface. It has no knowledge of which database is in use. The `DBReady func() bool` field is set to an atomic boolean that flips when `onReady()` fires, allowing `/healthz` to report the DB connection state without touching the store.

JSON response shapes match the existing `todolist-mariadb-go` app for UI compatibility:

- Create returns an array with one element: `[{"Id": "...", "Description": "...", "Completed": false}]`
- The `Id` field uses a capital `I` (matching the original UI's JavaScript expectations).

---

## Container Design

### Single-container model

The application and database run in the same container. This is an intentional design constraint:

- Mirrors a common Velero test pattern where one PVC backs one pod.
- Eliminates network dependencies between pods in the same test namespace.
- Simplifies Velero restore validation: restore the namespace, check that the pod comes up with data.

### Dockerfile structure

Both `Dockerfile.mariadb` and `Dockerfile.mongodb` use a two-stage build:

**Stage 1 (build):** `golang:1.25-alpine`
- CGO disabled, produces a fully static binary.
- Result copied to Stage 2 as `/opt/todolist/app`.

**Stage 2 (runtime):**
- `mariadb:latest` for the MariaDB image (includes `mariadbd`, `docker-entrypoint.sh`)
- `mongo:7` for the MongoDB image (includes `mongod`, `mongosh`)

### Permissions and arbitrary UID

OpenShift assigns a random UID from a namespace-specific range. The container must be writable by that UID. The pattern used in the Dockerfile is:

```dockerfile
RUN chown -R 1001:0 /opt/todolist /var/lib/mysql /tmp/log/todoapp && \
    chmod -R g=u    /opt/todolist /var/lib/mysql /tmp/log/todoapp
```

`chmod g=u` (group permissions equal user permissions) ensures that any process running as any UID in group 0 (GID 0, always granted by OpenShift) can read and write the same paths as the owner.

At runtime, `entrypoint.sh` re-applies `chown -R "$(id -u):0"` on the data directories before starting the database, so volume-mounted PVC paths are writable by the injected UID regardless of what the Dockerfile pre-set.

---

## entrypoint.sh Logic

```
# Step 1 — Determine DB_BACKEND (in priority order):
#   1. DB_BACKEND env var (highest precedence)
#   2. .db_backend marker file on the PV
#      - MariaDB: /var/lib/mysql/data/.db_backend
#      - MongoDB: /var/lib/mongodb/.db_backend
#   3. Default: mariadb

if MYSQL_HOST is external (non-localhost) [mariadb mode]:
    exec ./app   ← skip local DB

if MONGO_URI points to external host [mongodb mode]:
    exec ./app   ← skip local DB

# All-in-one mode:
fix permissions on data dir (chown + chmod g=u)
write .db_backend marker to PV (first startup only)

if mariadb:
    set DATADIR_ARG="--datadir=/var/lib/mysql/data"
    find and run: docker-entrypoint.sh mariadbd $DATADIR_ARG
                  (or mariadbd --user=<uid> $DATADIR_ARG as fallback)
    poll: mariadb -u <user> -p <pass> <db> -e "SELECT 1"
    → wait up to 120s for app user connectivity

if mongodb:
    mongod --dbpath /var/lib/mongodb --bind_ip 127.0.0.1 --fork
    poll: mongosh --eval "db.adminCommand({ping:1})"
    → wait up to 120s

exec /opt/todolist/app
```

Key implementation notes:

- **DB_BACKEND marker file**: On first startup, `entrypoint.sh` writes the detected backend name to a hidden file inside the data directory on the PV (`.db_backend`). On subsequent starts (including after a Velero restore), if `DB_BACKEND` is not set in the environment, the entrypoint reads this file to determine the correct backend. This makes the deployment resilient to restores where the env var was not captured in the backup.

- **MariaDB `--datadir` flag**: The official MariaDB `docker-entrypoint.sh` does **not** honour the `MARIADB_DATA_DIR` or `MYSQL_DATADIR` environment variables for changing the data directory. The `--datadir` CLI flag must be passed directly to `mariadbd`. Without this, MariaDB writes all data to `/var/lib/mysql` (the compiled-in default, which is ephemeral container storage), and data is lost on pod restart.

- **MariaDB startup delegation**: Startup delegates to the official `docker-entrypoint.sh` so that user/database creation from `MYSQL_*` env vars is handled correctly without duplicating that logic. A candidate list of known entrypoint paths is tried in order (RHEL `run-mysqld` first, then upstream image paths).

- **MariaDB readiness check**: Verifies the *application user* can connect (not just root), ensuring the Go app can immediately query the DB after `exec`.

- **MongoDB readiness check**: Uses the `mongosh` exit code rather than parsing output, because `mongosh` changed its output format in version 6+ (removing quotes from `{ ok: 1 }`).

---

## OpenShift Deployment

### Security Context Constraints

Each namespace gets a custom SCC (`mongo-persistent-scc` / `mysql-persistent-scc`) with `RunAsAny` for `runAsUser`, `fsGroup`, and `supplementalGroups`. This allows the container to run as any UID and ensures the PVC is group-writable.

The SCC grants access to the namespace's `ServiceAccount` via the `users` field on the SCC object:

```yaml
users:
- system:admin
- system:serviceaccount:mysql-persistent:mysql-persistent-sa
```

### Probes

All three probes use `/healthz`. The `/readyz` endpoint is available for manual health checks (it pings the DB and returns 503 until connected) but is not used by the Kubernetes probe configuration — doing so would cause the pod to be restarted during normal DB initialisation.

| Probe           | Endpoint  | Rationale                                                                    |
|-----------------|-----------|------------------------------------------------------------------------------|
| `startupProbe`  | `/healthz` | App is live immediately; `failureThreshold: 60` allows ~5 min for DB init  |
| `livenessProbe` | `/healthz` | Detects Go process hang                                                      |
| `readinessProbe`| `/healthz` | Prevents traffic before the Go app is fully up; DB readiness is enforced by `startupProbe` timeout |

### Volume mounts

| Backend  | Volume name  | Mount path            | PVC claim name |
|----------|--------------|-----------------------|----------------|
| MariaDB  | `mysql-data` | `/var/lib/mysql/data` | `mysql`        |
| MongoDB  | `mongo-data` | `/var/lib/mongodb`    | `mongo`        |

MariaDB data is stored at `/var/lib/mysql/data` (not the compiled-in default of `/var/lib/mysql`) so that the PVC is mounted directly at the datadir. The `--datadir` flag passed to `mariadbd` in `entrypoint.sh` enforces this.

---

## Velero / OADP Data Persistence

This section is critical — without the correct configuration Velero will back up the PVC *object* only, not its contents, and data will be lost after restore.

### Kopia file-system backup (recommended for KOPIA backup type)

Add the following annotation to the pod template in the Deployment so the Velero NodeAgent (Kopia) backs up the named volume:

```yaml
spec:
  template:
    metadata:
      annotations:
        backup.velero.io/backup-volumes: mysql-data   # or mongo-data
```

This annotation is already present in all manifests under `OPENSHIFT/mysql-persistent/` and `OPENSHIFT/mongo-persistent/`.

### CSI snapshot backup (for CSI backup type)

The VolumeSnapshotClass used by the storage class must carry the Velero label:

```bash
oc label volumesnapshotclass <name> velero.io/csi-volumesnapshot-class=true
```

On AWS with EBS CSI:

```bash
oc label volumesnapshotclass csi-aws-vsc velero.io/csi-volumesnapshot-class=true
```

Without this label, Velero's CSI plugin will not take a VolumeSnapshot during backup.

### Restore behaviour

When using Kopia, Velero injects a `restore-wait` init container into the restored pod. This init container restores the PV data before the main container (and therefore the database) starts. MariaDB's `docker-entrypoint.sh` detects the existing datadir (`mysql` system database present) and skips re-initialization, preserving all backed-up rows.

---

## Testing Strategy

### Tier 1 — Unit tests (no DB)

`internal/api/handlers_test.go` covers every HTTP handler using `MockStore`, an in-memory implementation of `TodoStore`. Tests are table-driven and cover happy paths, 400/404/503 error cases.

```bash
go test ./internal/... -v
```

### Tier 2 — Store integration tests (real DB, skipped by default)

`internal/store/mariadb/mariadb_test.go` and `internal/store/mongodb/mongodb_test.go` connect to a real database and exercise the full CRUD lifecycle. Skipped unless the relevant environment variable is set:

- `TEST_MYSQL_DSN=1` (plus `MYSQL_*` env vars) for MariaDB
- `TEST_MONGO_URI=mongodb://...` for MongoDB

### Tier 3 — HTTP smoke test (live deployment)

`test/smoke/smoke_test.go` is a Go test that skips unless `TEST_APP_URL` is set. Runs the full CRUD sequence against a live deployment and asserts HTTP status codes and response bodies.

```bash
TEST_APP_URL=http://todolist-route-mysql-persistent.apps.example.com \
  go test ./test/smoke/... -v
```

### Tier 4 — OpenShift smoke script

`test/openshift-smoke.sh` is a bash script for CI. It:
1. Polls `oc get pods` until the pod is `1/1 Running`.
2. Polls `/readyz` until the DB is connected.
3. Runs the full CRUD sequence using `curl`.

```bash
./test/openshift-smoke.sh mysql-persistent \
  http://todolist-route-mysql-persistent.apps.example.com
```

---

## Image Build and Registry

`build-manifest.sh` uses `podman` to build per-architecture images and then combines them into a multi-arch manifest list.

| Registry | Variable        | Default image name                                   | Expiry     |
|----------|-----------------|------------------------------------------------------|------------|
| ttl.sh   | `REGISTRY=ttl`  | `ttl.sh/oadp-ci-todo2-go-testing-mariadb-<uuid>:3h` | `TTL` env (default `3h`) |
| Quay     | `REGISTRY=quay` | `quay.io/migtools/oadp-ci-todo2-go-testing-mariadb:latest` | None |

Two images are produced per build:
- `*-mariadb-*` — built from `Dockerfile.mariadb`
- `*-mongodb-*` — built from `Dockerfile.mongodb`
