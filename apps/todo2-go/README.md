# todo2-go

A todo list web application written in Go that supports both **MariaDB** and **MongoDB** as the database backend. The application and database run inside a single container, making it ideal for testing Velero backup/restore operations on OpenShift.

---

## Quick Start

### Local — MariaDB

```bash
docker-compose -f docker-compose-mariadb.yml up --build
```

Open http://localhost:8000

### Local — MongoDB

```bash
docker-compose -f docker-compose.mongodb.yml up --build
```

Open http://localhost:8000

---

## Running Unit Tests

No database or running container needed.

```bash
cd apps/todo2-go
go test ./internal/... -v
```

### Store integration tests (optional)

MariaDB — requires a running MariaDB instance:

```bash
TEST_MYSQL_DSN=1 \
MYSQL_HOST=127.0.0.1 \
MYSQL_USER=changeme \
MYSQL_PASSWORD=changeme \
MYSQL_DATABASE=todolist \
go test ./internal/store/mariadb/... -v
```

MongoDB — requires a running MongoDB instance:

```bash
TEST_MONGO_URI=mongodb://127.0.0.1:27017 \
go test ./internal/store/mongodb/... -v
```

---

## Building Images

The `build-manifest.sh` script produces multi-arch (`linux/amd64`, `linux/arm64`) images using `podman`.

### Push to ttl.sh (default, expires in 3h)

```bash
./build-manifest.sh
```

Each run generates a unique tag suffix (UUID prefix). The image names are printed at the end, for example:

```
ttl.sh/oadp-ci-todo2-go-testing-mariadb-b4108e57:3h
ttl.sh/oadp-ci-todo2-go-testing-mongodb-b4108e57:3h
```

Customize the TTL:

```bash
TTL=24h ./build-manifest.sh
```

### Push to Quay

```bash
REGISTRY=quay ./build-manifest.sh
```

Produces:

```
quay.io/migtools/oadp-ci-todo2-go-testing-mariadb:latest
quay.io/migtools/oadp-ci-todo2-go-testing-mongodb:latest
```

---

## Deploying to OpenShift

### MariaDB

```bash
cd OPENSHIFT/mysql-persistent
oc create -f mysql-persistent-csi.yaml -f pvc/default_sc.yaml
```

### MongoDB

```bash
cd OPENSHIFT/mongo-persistent
oc create -f mongo-persistent-csi.yaml -f pvc/default_sc.yaml
```

Cloud-provider-specific PVC variants are in each `pvc/` subdirectory (aws, azure, gcp, ibmcloud, openstack).

After applying, check the pod:

```bash
oc get pods -n mysql-persistent   # or mongo-persistent
oc logs deployment/todolist -n mysql-persistent
```

The app exposes a route automatically. Get the URL:

```bash
oc get route -n mysql-persistent
```

---

## Smoke Tests

### HTTP smoke test (Go)

Works against any live deployment — local docker-compose or OpenShift route:

```bash
TEST_APP_URL=http://localhost:8000 go test ./test/smoke/... -v
# or
TEST_APP_URL=http://todolist-route-mysql-persistent.apps.example.com \
  go test ./test/smoke/... -v
```

### OpenShift smoke script

Polls pod readiness and runs CRUD assertions via `curl`:

```bash
./test/openshift-smoke.sh mysql-persistent \
  http://todolist-route-mysql-persistent.apps.example.com
```

---

## API Endpoints

| Method   | Path               | Description                             |
|----------|--------------------|-----------------------------------------|
| `GET`    | `/`                | Serve web UI (`index.html`)             |
| `GET`    | `/resources/*`     | Static assets (CSS, JS)                 |
| `GET`    | `/favicon.ico`     | Favicon                                 |
| `GET`    | `/healthz`         | Liveness: `{"alive":true,"db":"ready"}` |
| `GET`    | `/readyz`          | Readiness: 200 only when DB ping passes |
| `GET`    | `/log`             | Serve `/tmp/log/todoapp/app.log`        |
| `GET`    | `/todo-incomplete` | List incomplete items                   |
| `GET`    | `/todo-completed`  | List completed items                    |
| `GET`    | `/todo/{id}`       | Get a single item by ID                 |
| `POST`   | `/todo`            | Create item; form field: `description`  |
| `POST`   | `/todo/{id}`       | Update item; form field: `completed`    |
| `DELETE` | `/todo/{id}`       | Delete item                             |

---

## Environment Variables

| Variable              | Default       | Description                                                            |
|-----------------------|---------------|------------------------------------------------------------------------|
| `DB_BACKEND`          | `mariadb`     | `mariadb` or `mongodb`                                                 |
| `MYSQL_HOST`          | `127.0.0.1`   | MariaDB host; non-local value skips local DB startup                   |
| `MYSQL_PORT`          | `3306`        | MariaDB port                                                           |
| `MYSQL_USER`          | `changeme`    | DB username                                                            |
| `MYSQL_PASSWORD`      | `changeme`    | DB password                                                            |
| `MYSQL_ROOT_PASSWORD` | `root`        | Root password (used during DB initialisation only)                     |
| `MYSQL_DATABASE`      | `todolist`    | Database name                                                          |
| `MONGO_URI`           | _(unset)_     | Full MongoDB URI; if unset or points to 127.0.0.1, starts local mongod |
| `MONGO_DATABASE`      | `todolist`    | MongoDB database name                                                  |
| `APP_PORT`            | `8000`        | HTTP listen port                                                       |
| `LOG_LEVEL`           | `info`        | Logrus log level (`debug`, `info`, `warning`, `error`)                 |
| `STATIC_DIR`          | `web`         | Path to the directory containing `index.html` and `resources/`         |

---

## Directory Layout

```
apps/todo2-go/
├── cmd/server/main.go              # Entry point
├── internal/
│   ├── model/todo.go               # TodoItem struct
│   ├── store/
│   │   ├── store.go                # TodoStore interface + sentinel errors
│   │   ├── mariadb/mariadb.go      # GORM/MariaDB implementation
│   │   ├── mariadb/mariadb_test.go # MariaDB integration test
│   │   ├── mongodb/mongodb.go      # mongo-driver implementation
│   │   └── mongodb/mongodb_test.go # MongoDB integration test
│   └── api/
│       ├── handlers.go             # HTTP handlers
│       ├── handlers_test.go        # Unit tests (mock store)
│       └── mock_store.go           # In-memory mock TodoStore
├── web/                            # Static UI assets
├── scripts/entrypoint.sh           # Container startup script
├── OPENSHIFT/
│   ├── mysql-persistent/           # MariaDB OpenShift manifests
│   └── mongo-persistent/           # MongoDB OpenShift manifests
├── test/
│   ├── smoke/smoke_test.go         # HTTP smoke test (Go)
│   └── openshift-smoke.sh          # OpenShift smoke script (bash)
├── Dockerfile.mariadb
├── Dockerfile.mongodb
├── docker-compose-mariadb.yml
├── docker-compose.mongodb.yml
├── build-manifest.sh
└── go.mod
```
