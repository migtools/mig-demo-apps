# Todolist MongoDB Go — Single Container (UBI 9)

A Go-based todolist application with a MongoDB backend, packaged into a
**single UBI-9 container**. Both `mongod` and the Go web application run
inside the same container, eliminating race conditions between separate
database and application pods.

## Architecture

```
┌──────────────────────────────────────┐
│  Container (UBI 9 + MongoDB 7.0)     │
│                                      │
│  entrypoint.sh                       │
│    ├─ mongod --auth (background)     │
│    └─ todolist Go app (foreground)   │
│                                      │
│  mongod listens on 127.0.0.1:27017   │
│  Go app  listens on 0.0.0.0:8000    │
│                                      │
│  PVC mounted at /data/db             │
└──────────────────────────────────────┘
```

## Local Development

Build and run everything with docker-compose:

```bash
docker-compose up -d --build
```

The app will be available at http://localhost:8000

To stop:

```bash
docker-compose down
```

Data persists in a named volume (`mongodata`). To wipe and start fresh:

```bash
docker-compose down -v
```

## Building Container Images

### Using podman (multi-arch):

```bash
./build.sh        # build locally
./build.sh -p     # build and push to quay.io
```

### Using docker buildx:

```bash
make manifest-buildx
```

### Using podman manifest:

```bash
make manifest-docker
```

Override the registry/image/version:

```bash
make manifest-buildx REGISTRY=quay.io/myuser IMAGE=todolist-mongo VERSION=v2
```

## OpenShift Deployment

### Standard (filesystem PVC):

```bash
oc apply -f OPENSHIFT/mongo-persistent.yaml
```

### CSI storage:

```bash
oc apply -f OPENSHIFT/mongo-persistent-csi.yaml -f OPENSHIFT/pvc/aws.yaml
```

### Block storage:

```bash
oc apply -f OPENSHIFT/mongo-persistent-block.yaml -f OPENSHIFT/pvc/aws-block-mode.yaml
```

## Testing

See the `test/` directory for the test suite:

```bash
cd test
pip install -r requirements.txt
python run_tests.py --url http://localhost:8000
```

Or use the quick curl-based smoke tests:

```bash
bash test/curl_tests.sh
```

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `MONGO_INITDB_ROOT_USERNAME` | `changeme` | MongoDB admin username |
| `MONGO_INITDB_ROOT_PASSWORD` | `changeme` | MongoDB admin password |
| `MONGO_INITDB_DATABASE` | `todolist` | Database name |

## API Endpoints

| Method | Path | Description |
|---|---|---|
| GET | `/` | Web UI |
| GET | `/healthz` | Health check (pings MongoDB) |
| GET | `/todo-completed` | List completed items |
| GET | `/todo-incomplete` | List incomplete items |
| POST | `/todo` | Create item (`description` form field) |
| POST | `/todo/{id}` | Update item (`completed` form field) |
| DELETE | `/todo/{id}` | Delete item |
| GET | `/log` | Application log file |

## Notes

* Originally based on https://github.com/sdil/learning/blob/master/go/todolist-mysql-go/todolist.go
* OADP (OpenShift API for Data Protection) demo application
* Velero backup hooks (fsyncLock/fsyncUnlock) are configured in the OpenShift manifests
