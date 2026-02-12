# todolist-mariadb-go

A sample todolist application for OADP (OpenShift API for Data Protection) e2e testing.

* Originally based on:
https://github.com/sdil/learning/blob/master/go/todolist-mysql-go/todolist.go

## Architecture

This application runs as a **single container** based on `registry.redhat.io/rhel9/mariadb-1011`.
Both the MariaDB database and the Go todolist application run inside the same container,
managed by an entrypoint script (`scripts/entrypoint.sh`).

This eliminates the race condition that occurred with the previous two-container architecture
where the todolist app could fail to connect to the database after a Velero restore.

### How it works

1. The entrypoint script starts MariaDB in the background using the image's built-in `run-mysqld`
2. It waits for MariaDB to accept connections
3. It starts the Go todolist application in the foreground
4. The Go app connects to MariaDB at `127.0.0.1:3306` with retry logic

Database initialization (creating the database and user) is handled automatically by
the `rhel9/mariadb-1011` image via environment variables:
- `MYSQL_USER` / `MYSQL_PASSWORD` -- application DB credentials
- `MYSQL_ROOT_PASSWORD` -- root password
- `MYSQL_DATABASE` -- database name to create

### MariaDB password settings

The DSN used by the Go application:
```
dsn := "changeme:changeme@tcp(127.0.0.1:3306)/todolist?charset=utf8mb4&parseTime=True&loc=Local"
```

## Local Setup with docker-compose

```
docker-compose up
```

Navigate your browser to:
 * http://localhost:8000

## Deploy to OpenShift

```
oc create -f mysql-persistent.yaml
```
Or with CSI storage:
```
oc create -f mysql-persistent-csi.yaml -f pvc/$cloud.yaml
```

## Testing

There are some basic curl and python tests in the tests directory where you can
see the API is exercised and the database is populated.
```
cd test
python test.py
```

## Building

Build a new container:
```
podman build -t quay.io/migtools/oadp-ci-todolist-mariadb-go:latest .
podman push quay.io/migtools/oadp-ci-todolist-mariadb-go:latest
```

### Build for multi-arch

Using the Makefile:
```
make manifest-buildx REGISTRY=quay.io/migtools IMAGE=oadp-ci-todolist-mariadb-go VERSION=latest
```

See also: https://developers.redhat.com/articles/2023/11/03/how-build-multi-architecture-container-images

## Build a VM with the todolist installed directly (without containers)

* Note: this was tested with Fedora 39
    * Fedora-Cloud-Base-39-1.5.x86_64.qcow2

* Get a RHEL, CentOS, or Fedora VM image.
 * Copy the qcow2 to /var/lib/libvirt/images
 * Update the cloud-init/todolist-data file with your public ssh keys
 ```
 ssh-authorized-keys:
    - ssh-ed25519 AAAAC3... your-key
 ```
 * Copy the cloud-init/todolist-data to /var/lib/libvirt/boot/cloud-init/

* Sample virt-install:
```
sudo virt-install --name todolist-mariadb-1  --memory memory=3072  --cpu host --vcpus 2  --graphics none  --os-variant fedora39  --import  --disk /var/lib/libvirt/images/Fedora-Cloud-Base-39-1.5.x86_64.qcow2,format=qcow2,bus=virtio  --disk size=8 --network type=network,source=default,model=virtio  --cloud-init user-data=/var/lib/libvirt/boot/cloud-init/todolist-data
```

* Wait for both the install and cloud-init to finish.
* Browse to the VM IP and port: `http://192.168.122.113:8000/` for example

* If the cloud-init fails, test with:
```
sudo cloud-init schema --system
```

## Notes
* The app will NOT create prepopulated items in the todo list at startup.
