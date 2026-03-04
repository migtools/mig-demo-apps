#!/bin/bash
set -e

# Push destination: ttl (default) or quay
#   REGISTRY=ttl  - push to ttl.sh (default), requires TTL env (default 3h)
#   REGISTRY=quay - push to quay.io
if [[ -z $REGISTRY ]]; then
    REGISTRY=ttl
fi
if [[ "$REGISTRY" != "quay" && "$REGISTRY" != "ttl" ]]; then
    echo "REGISTRY must be 'ttl' or 'quay' (got: $REGISTRY)"
    exit 1
fi

# TTL for ttl.sh expiry tag (default 3h)
if [[ -z $TTL ]]; then
    TTL=3h
fi

# Default manifests
QUAY_MARIADB="quay.io/migtools/oadp-ci-todo2-go-testing-mariadb:latest"
QUAY_MONGODB="quay.io/migtools/oadp-ci-todo2-go-testing-mongodb:latest"
if [[ $REGISTRY == ttl ]]; then
    TTL_TAG=$(uuidgen | head -c 8)
    TTL_MARIADB="ttl.sh/oadp-ci-todo2-go-testing-mariadb-${TTL_TAG}:${TTL}"
    TTL_MONGODB="ttl.sh/oadp-ci-todo2-go-testing-mongodb-${TTL_TAG}:${TTL}"
    echo "Pushing to ttl.sh: $TTL_MARIADB and $TTL_MONGODB (expires in $TTL)"
fi

if [[ -z $PLATFORM_LIST ]]; then
    PLATFORM_LIST="linux/amd64,linux/arm64"
fi

build_and_push() {
    local dockerfile=$1
    local manifest_name=$2
    local image_list=""
    for PLATFORM in $(echo $PLATFORM_LIST | tr "," " "); do
        echo "Building $dockerfile for platform: $PLATFORM"
        ARCH="${PLATFORM#*/}"
        # Tag per-arch as manifest_name-arch (e.g. ttl.sh/name:3h-amd64)
        IMAGE="${manifest_name}-${ARCH}"
        podman build --platform=$PLATFORM -f "$dockerfile" -t "$IMAGE" .
        podman push "$IMAGE"
        image_list="$image_list $IMAGE"
    done
    echo "Creating manifest $manifest_name from: $image_list"
    podman manifest rm "$manifest_name" 2>/dev/null || true
    podman manifest create "$manifest_name" $image_list
    podman manifest push "$manifest_name"
}

# MariaDB image (Dockerfile)
if [[ $REGISTRY == ttl ]]; then
    BASE_MARIADB=$TTL_MARIADB
else
    BASE_MARIADB=$QUAY_MARIADB
fi
build_and_push Dockerfile.mariadb "$BASE_MARIADB"

# MongoDB image (Dockerfile.mongodb)
if [[ $REGISTRY == ttl ]]; then
    BASE_MONGODB=$TTL_MONGODB
else
    BASE_MONGODB=$QUAY_MONGODB
fi
build_and_push Dockerfile.mongodb "$BASE_MONGODB"

echo "Done. MariaDB image: $BASE_MARIADB  MongoDB image: $BASE_MONGODB"
