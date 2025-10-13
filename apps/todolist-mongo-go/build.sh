#!/bin/bash

VERSION_TAG=latest
REGISTRY=ttl.sh
USER=migtools
# Manifest name
MANIFEST_NAME="quay.io/migtools/oadp-ci-todolist-mongo-go-3"
#MANIFEST_NAME="ttl.sh/oadp-ci-todolist-mongo-go-3"

if [[ -z "${VERSION_TAG:+x}" ]]; then
    echo "Please set up VERSION_TAG variable"
    exit 2
fi

if [[ -z "${REGISTRY:+x}" ]]; then
  echo "Please set up REGISTRY variable"
  exit 2
fi

if [[ -z "${USER:+x}" ]]; then
  echo "Please set up USER variable"
  exit 2
fi

# Publish flag
if [[ "$#" -eq 1 ]] && [[ "$1" == "-p" ]]; then
  SHOULD_PUBLISH=1
fi

REGISTRY="$REGISTRY"
USER="$USER"
IMAGE_NAME="todolist-mongo-go"
IMAGE_TAG="${VERSION_TAG}"

# Base image name
BASE_IMAGE_NAME="${MANIFEST_NAME}"
# Create a multi-architecture manifest (remove if exists)
podman manifest rm ${MANIFEST_NAME}:${VERSION_TAG} 2>/dev/null || true
podman manifest create ${MANIFEST_NAME}:${VERSION_TAG}

for arch in arm64 amd64; do
    ARCH_FLAGS=("--arch" "${arch}")
    if [[ "$arch" = "arm64" ]]; then
        ARCH_FLAGS=("${ARCH_FLAGS[@]}" "--variant" "v8")
    fi

    echo "building: $BASE_IMAGE_NAME-$arch"
    podman build \
      -t "$MANIFEST_NAME-$arch" \
      --manifest "${MANIFEST_NAME}:${VERSION_TAG}" \
      --arch $arch \
      -f Dockerfile
    
    echo "========= IMAGE INFO ===============" 
    podman image inspect "$BASE_IMAGE_NAME-$arch" --format "{{.Architecture}}"
    echo "========= END IMAGE INFO ===============" 
done

# Publish images to the registry
if [[ "$SHOULD_PUBLISH" -eq 1 ]]; then
  podman manifest push --all "${MANIFEST_NAME}:${VERSION_TAG}" \
    "docker://${MANIFEST_NAME}:${VERSION_TAG}"
fi

podman manifest inspect "${MANIFEST_NAME}:${VERSION_TAG}"
