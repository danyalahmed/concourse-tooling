IMAGE_REGISTRY ?= ghcr.io/danyalahmed/concourse-tooling

# Single-binary resources (one Dockerfile, one binary each)
SINGLE_IMAGES := smb-resource cpanel-db-backup-resource cron-resource

# Multi-command resources (one image per subcommand)
RESTIC_CMDS := backup prune restore stats

# ── Phony targets ─────────────────────────────────────────────────────────────

.PHONY: all build test vet fmt clean help

.PHONY: $(addprefix image-,$(SINGLE_IMAGES))
.PHONY: $(addprefix push-,$(SINGLE_IMAGES))
.PHONY: $(addprefix image-push-,$(SINGLE_IMAGES))

.PHONY: $(addprefix image-restic-,$(RESTIC_CMDS))
.PHONY: $(addprefix push-restic-,$(RESTIC_CMDS))
.PHONY: $(addprefix image-push-restic-,$(RESTIC_CMDS))
.PHONY: image-restic push-restic image-push-restic

.PHONY: image push image-push

# ── Default ──────────────────────────────────────────────────────────────────

all: vet fmt build

# ── Go build (all modules) ───────────────────────────────────────────────────

build:
	go build ./...

# ── Single-binary images ─────────────────────────────────────────────────────

image: $(addprefix image-,$(SINGLE_IMAGES)) image-restic

define SINGLE_IMAGE_TARGET
image-$(1):
	podman build -f $(1)/Dockerfile -t $(IMAGE_REGISTRY)/$(1) .
endef
$(foreach img,$(SINGLE_IMAGES),$(eval $(call SINGLE_IMAGE_TARGET,$(img))))

push: $(addprefix push-,$(SINGLE_IMAGES)) push-restic

define SINGLE_PUSH_TARGET
push-$(1):
	podman push $(IMAGE_REGISTRY)/$(1)
endef
$(foreach img,$(SINGLE_IMAGES),$(eval $(call SINGLE_PUSH_TARGET,$(img))))

image-push: $(addprefix image-push-,$(SINGLE_IMAGES)) image-push-restic

define SINGLE_IMAGE_PUSH_TARGET
image-push-$(1): image-$(1) push-$(1)
	@true
endef
$(foreach img,$(SINGLE_IMAGES),$(eval $(call SINGLE_IMAGE_PUSH_TARGET,$(img))))

# ── Restic multi-command images ──────────────────────────────────────────────

image-restic: $(addprefix image-restic-,$(RESTIC_CMDS))

define RESTIC_IMAGE_TARGET
image-restic-$(1):
	podman build \
		--build-arg TARGET_CMD=$(1) \
		-f restic-resource/Dockerfile \
		-t $(IMAGE_REGISTRY)/cpanel-smb-$(1)-resource \
		.
endef
$(foreach cmd,$(RESTIC_CMDS),$(eval $(call RESTIC_IMAGE_TARGET,$(cmd))))

push-restic: $(addprefix push-restic-,$(RESTIC_CMDS))

define RESTIC_PUSH_TARGET
push-restic-$(1):
	podman push $(IMAGE_REGISTRY)/cpanel-smb-$(1)-resource
endef
$(foreach cmd,$(RESTIC_CMDS),$(eval $(call RESTIC_PUSH_TARGET,$(cmd))))

image-push-restic: $(addprefix image-push-restic-,$(RESTIC_CMDS))

define RESTIC_IMAGE_PUSH_TARGET
image-push-restic-$(1): image-restic-$(1) push-restic-$(1)
	@true
endef
$(foreach cmd,$(RESTIC_CMDS),$(eval $(call RESTIC_IMAGE_PUSH_TARGET,$(cmd))))

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
	@echo '    build               go build ./... (all modules)'
	@echo ''
	@echo '  Single-binary images (one Dockerfile each):'
	@echo '    image               Build all Docker images'
	@echo '    image-<name>        Build a single image  (e.g. image-smb-resource)'
	@echo '    push                Push all Docker images'
	@echo '    push-<name>         Push a single image   (e.g. push-smb-resource)'
	@echo '    image-push          Build + push all images'
	@echo '    image-push-<name>   Build + push a single image'
	@echo ''
	@echo '  Restic multi-command images:'
	@echo '    image-restic              Build all restic images'
	@echo '    image-restic-<cmd>        Build a single restic image  (e.g. image-restic-backup)'
	@echo '    push-restic               Push all restic images'
	@echo '    push-restic-<cmd>         Push a single restic image   (e.g. push-restic-backup)'
	@echo '    image-push-restic         Build + push all restic images'
	@echo '    image-push-restic-<cmd>   Build + push a single restic image'
	@echo ''
	@echo '  Quality:'
	@echo '    test                Run go test ./...'
	@echo '    vet                 Run go vet ./...'
	@echo '    fmt                 Run go fmt ./...'
	@echo '    clean               Remove bin/'
	@echo ''
	@echo 'Single-binary images: $(SINGLE_IMAGES)'
	@echo 'Restic commands:      $(RESTIC_CMDS)'
	@echo 'Image registry:       $(IMAGE_REGISTRY)'
	@echo ''
	@echo 'Examples:'
	@echo '  make image-smb-resource'
	@echo '    podman build -f smb-resource/Dockerfile -t $(IMAGE_REGISTRY)/smb-resource .'
	@echo ''
	@echo '  make image-restic-backup'
	@echo '    podman build --build-arg TARGET_CMD=backup -f restic-resource/Dockerfile -t $(IMAGE_REGISTRY)/cpanel-smb-backup-resource .'
	@echo ''
	@echo '  make push-restic-stats'
	@echo '    podman push $(IMAGE_REGISTRY)/cpanel-smb-stats-resource'
	@echo ''
	@echo '  make image-push-cron'
	@echo '    (build then push in one go)'
