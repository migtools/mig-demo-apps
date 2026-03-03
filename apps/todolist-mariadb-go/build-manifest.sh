#!/bin/bash
set -e

# Default manifest name
if [[ -z $MANIFEST ]]; then
    MANIFEST="quay.io/migtools/oadp-ci-todolist-mariadb-go-testing:testing"
fi

# Default platforms: amd64 and arm64
if [[ -z $PLATFORM_LIST ]]; then
    PLATFORM_LIST="linux/amd64,linux/arm64"
fi

# Optional: push to ttl.sh ephemeral registry for testing
# Usage: TTL=2h ./build-manifest.sh
if [[ -n $TTL ]]; then
    TTL_TAG=$(uuidgen | head -c 8)
    TTL_MANIFEST="ttl.sh/todolist-mariadb-go-${TTL_TAG}:${TTL}"
    echo "Will also push to ttl.sh: $TTL_MANIFEST"
fi

IMAGE_LIST=""

for PLATFORM in $(echo $PLATFORM_LIST | tr "," " "); do
    echo "Building container image for platform: $PLATFORM"
    ARCH="${PLATFORM#*/}"
    IMAGE=$MANIFEST-$ARCH
    podman build --platform=$PLATFORM -t $IMAGE .
    podman push $IMAGE
    IMAGE_LIST="$IMAGE_LIST $IMAGE"
done

echo "Creating manifest $MANIFEST from images: $IMAGE_LIST"
podman manifest create $MANIFEST $IMAGE_LIST
podman manifest push $MANIFEST
echo "Pushed: $MANIFEST"

# Push to ttl.sh if requested
if [[ -n $TTL_MANIFEST ]]; then
    echo "Creating ttl.sh manifest: $TTL_MANIFEST"
    podman manifest create $TTL_MANIFEST $IMAGE_LIST
    podman manifest push $TTL_MANIFEST
    echo "Pushed to ttl.sh: $TTL_MANIFEST (expires in $TTL)"
fi
