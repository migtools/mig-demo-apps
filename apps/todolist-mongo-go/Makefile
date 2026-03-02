REGISTRY ?= quay.io/migtools
IMAGE ?= oadp-ci-todolist-mongo-go
VERSION ?= latest

MANIFEST_IMG = $(REGISTRY)/$(IMAGE):$(VERSION)

PLATFORMS ?= linux/amd64,linux/arm64

ifeq ($(shell docker buildx inspect 2>/dev/null | awk '/Status/ { print $$2 }'), running)
	BUILDX_ENABLED ?= true
else
	BUILDX_ENABLED ?= false
endif

define BUILDX_ERROR
buildx not enabled, refusing to run this recipe
endef

manifest-buildx:
ifneq ($(BUILDX_ENABLED), true)
	$(error $(BUILDX_ERROR))
endif
	@echo "Running $@ target for platforms: $(PLATFORMS)"
	docker buildx build \
	--push \
	--platform $(PLATFORMS) \
	-t $(MANIFEST_IMG) \
	.

manifest-docker:
	@echo "Running $@ target for platforms: $(PLATFORMS)"
	MANIFEST="$(MANIFEST_IMG)" \
	PLATFORM_LIST=$(PLATFORMS) \
	./build-manifest.sh
