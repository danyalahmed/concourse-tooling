IMAGE_REGISTRY ?= ghcr.io/danyalahmed/concourse-tooling
CMD_NAMES     := backup prune restore stats

.PHONY: all build test vet fmt clean help

.PHONY: $(addprefix build-,$(CMD_NAMES))
.PHONY: $(addprefix image-,$(CMD_NAMES))
.PHONY: $(addprefix push-,$(CMD_NAMES))
.PHONY: $(addprefix image-push-,$(CMD_NAMES))
.PHONY: push image-push

# ── Default ──────────────────────────────────────────────────────────────────

all: vet fmt build

# ── Go build (standalone binaries) ───────────────────────────────────────────

build: $(addprefix build-,$(CMD_NAMES))

define BUILD_target
build-$(1):
	go build -o bin/$(1) ./cmd/$(1)/
endef
$(foreach cmd,$(CMD_NAMES),$(eval $(call BUILD_target,$(cmd))))

# ── Docker images (via podman, context = repo root) ──────────────────────────

image: $(addprefix image-,$(CMD_NAMES))

define IMAGE_target
image-$(1):
	podman build \
		--build-arg TARGET_CMD=$(1) \
		-f restic-resource/Dockerfile \
		-t $(IMAGE_REGISTRY)/cpanel-smb-$(1)-resource \
		.
endef
$(foreach cmd,$(CMD_NAMES),$(eval $(call IMAGE_target,$(cmd))))

# ── Push images ──────────────────────────────────────────────────────────────

push: $(addprefix push-,$(CMD_NAMES))

define PUSH_target
push-$(1):
	podman push $(IMAGE_REGISTRY)/cpanel-smb-$(1)-resource
endef
$(foreach cmd,$(CMD_NAMES),$(eval $(call PUSH_target,$(cmd))))

# ── Build + push (combined) ──────────────────────────────────────────────────

image-push: $(addprefix image-push-,$(CMD_NAMES))

define IMAGE_PUSH_target
image-push-$(1): image-$(1) push-$(1)
	@true
endef
$(foreach cmd,$(CMD_NAMES),$(eval $(call IMAGE_PUSH_target,$(cmd))))

# ── Quality ──────────────────────────────────────────────────────────────────

test:
	go test ./...

vet:
	go vet ./...

fmt:
	go fmt ./...

# ── Clean ────────────────────────────────────────────────────────────────────

clean:
	rm -rf bin/

# ── Help ─────────────────────────────────────────────────────────────────────

help:
	@echo 'Usage: make <target>'
	@echo ''
	@echo '  Build:'
	@echo '    build           Build all binaries  (go build ./cmd/*/)'
	@echo '    build-<cmd>     Build a single binary  (e.g. build-backup)'
	@echo ''
	@echo '  Images:'
	@echo '    image           Build all Docker images'
	@echo '    image-<cmd>     Build a single Docker image  (e.g. image-stats)'
	@echo '    push            Push all Docker images'
	@echo '    push-<cmd>      Push a single Docker image  (e.g. push-stats)'
	@echo '    image-push      Build + push all Docker images'
	@echo '    image-push-<cmd> Build + push a single Docker image'
	@echo ''
	@echo '  Quality:'
	@echo '    test            Run go test ./...'
	@echo '    vet             Run go vet ./...'
	@echo '    fmt             Run go fmt ./...'
	@echo '    clean           Remove bin/'
	@echo ''
	@echo 'Available cmds: $(CMD_NAMES)'
	@echo 'Image registry: $(IMAGE_REGISTRY)'
	@echo ''
	@echo 'Examples:'
	@echo '  make image-stats'
	@echo '    podman build --build-arg TARGET_CMD=stats -f restic-resource/Dockerfile -t $(IMAGE_REGISTRY)/cpanel-smb-stats-resource .'
	@echo ''
	@echo '  make push-stats'
	@echo '    podman push $(IMAGE_REGISTRY)/cpanel-smb-stats-resource'
	@echo ''
	@echo '  make image-push-stats'
	@echo '    (build then push in one go)'
