#!/bin/bash
set -e

# Push destination: quay (default) or ttl
#   REGISTRY=quay  - push to quay.io only (default)
#   REGISTRY=ttl   - push to ttl.sh only (requires TTL, e.g. TTL=2h)
if [[ -z $REGISTRY ]]; then
    REGISTRY=quay
fi
if [[ "$REGISTRY" != "quay" && "$REGISTRY" != "ttl" ]]; then
    echo "REGISTRY must be 'quay' or 'ttl' (got: $REGISTRY)"
    exit 1
fi

# Default manifest name (quay)
if [[ -z $MANIFEST ]]; then
    MANIFEST="quay.io/migtools/oadp-ci-todolist-mariadb-go-testing:testing"
fi

# Default platforms: amd64 and arm64
if [[ -z $PLATFORM_LIST ]]; then
    PLATFORM_LIST="linux/amd64,linux/arm64"
fi

# ttl.sh: require TTL when REGISTRY=ttl; optional "also push" when REGISTRY=quay
if [[ $REGISTRY == ttl ]]; then
    if [[ -z $TTL ]]; then
        echo "When REGISTRY=ttl, TTL must be set (e.g. TTL=2h ./build-manifest.sh)"
        exit 1
    fi
    TTL_TAG=$(uuidgen | head -c 8)
    TTL_MANIFEST="ttl.sh/todolist-mariadb-go-${TTL_TAG}:${TTL}"
    echo "Pushing to ttl.sh only: $TTL_MANIFEST"
elif [[ -n $TTL ]]; then
    TTL_TAG=$(uuidgen | head -c 8)
    TTL_MANIFEST="ttl.sh/todolist-mariadb-go-${TTL_TAG}:${TTL}"
    echo "Will also push to ttl.sh: $TTL_MANIFEST"
fi

IMAGE_LIST=""

for PLATFORM in $(echo $PLATFORM_LIST | tr "," " "); do
    echo "Building container image for platform: $PLATFORM"
    ARCH="${PLATFORM#*/}"
    if [[ $REGISTRY == ttl ]]; then
        IMAGE=$TTL_MANIFEST-$ARCH
    else
        IMAGE=$MANIFEST-$ARCH
    fi
    podman build --platform=$PLATFORM -t $IMAGE .
    podman push $IMAGE
    IMAGE_LIST="$IMAGE_LIST $IMAGE"
done

if [[ $REGISTRY == ttl ]]; then
    echo "Creating manifest $TTL_MANIFEST from images: $IMAGE_LIST"
    podman manifest create $TTL_MANIFEST $IMAGE_LIST
    podman manifest push $TTL_MANIFEST
    echo "Pushed to ttl.sh: $TTL_MANIFEST (expires in $TTL)"
else
    echo "Creating manifest $MANIFEST from images: $IMAGE_LIST"
    podman manifest create $MANIFEST $IMAGE_LIST
    podman manifest push $MANIFEST
    echo "Pushed: $MANIFEST"
fi

# Push to ttl.sh as well if REGISTRY=quay and TTL was set
if [[ $REGISTRY == quay && -n $TTL_MANIFEST ]]; then
    echo "Creating ttl.sh manifest: $TTL_MANIFEST"
    podman manifest create $TTL_MANIFEST $IMAGE_LIST
    podman manifest push $TTL_MANIFEST
    echo "Pushed to ttl.sh: $TTL_MANIFEST (expires in $TTL)"
fi
