# CloseLoopAutomous — arms server build & release
# Go module lives in ./arms; import path is github.com/morpheumstreet/CloseLoopAutomous/arms (monorepo root + /arms).
# Git tags are at repo root.
# Use GOWORK=off if this repo is outside your go.work.
# Version and commit are injected at build time from git.

ARMS_DIR := arms
BINARY   := $(ARMS_DIR)/bin/arms
MAIN     := ./cmd/arms
# Default config for `make run` (override: make run CONFIG=config/arms.local.toml)
CONFIG   ?= config/arms.toml
PKG      := github.com/morpheumstreet/CloseLoopAutomous/arms/cmd/arms
VERSION  := $(shell git -C $(ARMS_DIR)/.. describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT   := $(shell git -C $(ARMS_DIR)/.. rev-parse --short HEAD 2>/dev/null || echo "")
LDFLAGS  := -ldflags "-X $(PKG).Version=$(VERSION) -X $(PKG).Commit=$(COMMIT)"
GOBIN    := $(shell go env GOPATH)/bin
CGO_ENABLED ?= 0

# Cross-build matrix (override from the command line if needed).
# Examples:
#   make dist TARGETS="darwin/arm64 linux/amd64"
#   git tag v1.2.3 && make dist   # VERSION string comes from git describe
TARGETS ?= darwin/arm64 darwin/amd64 linux/arm64 linux/amd64 windows/amd64

# Tagging: explicit TAG=v1.2.3 or semver bump (patch | minor | major).
TAG  ?=
BUMP ?= patch

.PHONY: build build-all clean clean-all install dist run
.PHONY: tag tag-push release upload-build-all upload-build-all-gz

# Alias for papercli-style workflows
build-all: dist

build:
	@mkdir -p $(ARMS_DIR)/bin
	cd $(ARMS_DIR) && GOWORK=off CGO_ENABLED=$(CGO_ENABLED) go build -trimpath $(LDFLAGS) -o bin/arms $(MAIN)

install: build
	@mkdir -p $(GOBIN)
	cp $(BINARY) $(GOBIN)/arms

# Run from repo root so paths in CONFIG (e.g. ./data/arms.db) match config/arms.toml
run: build
	@mkdir -p data
	GOWORK=off $(BINARY) -c $(CONFIG)

# Build gzipped binaries for all TARGETS into ./arms/bin
# Output naming:
#   arms/bin/arms_<version>_<os>_<arch>[.exe]
#   arms/bin/arms_<version>_<os>_<arch>[.exe].gz
dist:
	@mkdir -p $(ARMS_DIR)/bin
	@set -e; \
	for t in $(TARGETS); do \
		os="$${t%/*}"; \
		arch="$${t#*/}"; \
		ext=""; \
		if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
		base="arms_$(VERSION)_$${os}_$${arch}$${ext}"; \
		out="$(ARMS_DIR)/bin/$$base"; \
		echo "building $$os/$$arch -> $$out"; \
		GOWORK=off CGO_ENABLED=$(CGO_ENABLED) GOOS="$$os" GOARCH="$$arch" \
			go build -C $(ARMS_DIR) -trimpath $(LDFLAGS) -o "bin/$$base" $(MAIN); \
		echo "gzipping $$out -> $$out.gz"; \
		gzip -c "$$out" > "$$out.gz" && rm -f "$$out"; \
	done

# Create annotated semver tag on HEAD (local only). Use TAG= or BUMP=patch|minor|major.
tag:
	@set -e; \
	repo=$$(cd $(ARMS_DIR)/.. && pwd); \
	if [ -n "$(TAG)" ]; then \
	  new="$(TAG)"; \
	else \
	  last=$$(git -C "$$repo" describe --tags --abbrev=0 --match 'v*' 2>/dev/null || echo "v0.0.0"); \
	  last=$${last#v}; \
	  major=$$(echo "$$last" | cut -d. -f1); \
	  minor=$$(echo "$$last" | cut -d. -f2); \
	  patch=$$(echo "$$last" | cut -d. -f3 | cut -d- -f1); \
	  major=$${major:-0}; minor=$${minor:-0}; patch=$${patch:-0}; \
	  case "$(BUMP)" in \
	    major) major=$$((major + 1)); minor=0; patch=0 ;; \
	    minor) minor=$$((minor + 1)); patch=0 ;; \
	    patch) patch=$$((patch + 1)) ;; \
	    *) echo "error: BUMP must be patch, minor, or major (got $(BUMP))" >&2; exit 2 ;; \
	  esac; \
	  new="v$$major.$$minor.$$patch"; \
	fi; \
	case "$$new" in v*) ;; *) new="v$$new";; esac; \
	if git -C "$$repo" rev-parse "$$new" >/dev/null 2>&1; then \
	  echo "error: tag $$new already exists" >&2; exit 1; \
	fi; \
	git -C "$$repo" tag -a "$$new" -m "Release $$new"; \
	echo "tagged $$new (local)"

tag-push: tag
	@repo=$$(cd $(ARMS_DIR)/.. && pwd); \
	new=$$(git -C "$$repo" describe --tags --abbrev=0 --match 'v*'); \
	git -C "$$repo" push origin "$$new"

# Bump semver tag, push it, then produce gzipped cross-builds in ./arms/bin
release: tag-push dist

# Upload build artifacts from dist to Filebase IPFS and append results to uploaded.json.
# Requires FILEBASE_IPFS_API_KEY in env (e.g. FILEBASE_IPFS_API_KEY=... make upload-build-all).
#
# Uploads ONLY the gzipped artifacts for each target.
# Add scripts/filebase-ipfs-upload.sh to this repo if you use this (same idea as papercli).

upload-build-all: build-all
	@set -e; \
	if [ -z "$$FILEBASE_IPFS_API_KEY" ]; then \
		echo "error: FILEBASE_IPFS_API_KEY is required" >&2; \
		echo "hint: FILEBASE_IPFS_API_KEY=... make upload-build-all" >&2; \
		exit 2; \
	fi; \
	repo=$$(cd $(ARMS_DIR)/.. && pwd); \
	if [ ! -f "$$repo/scripts/filebase-ipfs-upload.sh" ]; then \
		echo "error: missing scripts/filebase-ipfs-upload.sh" >&2; \
		exit 2; \
	fi; \
	files=""; \
	for t in $(TARGETS); do \
		os="$${t%/*}"; \
		arch="$${t#*/}"; \
		ext=""; \
		if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
		out="$(ARMS_DIR)/bin/arms_$(VERSION)_$${os}_$${arch}$${ext}"; \
		files="$$files $$out.gz"; \
	done; \
	bash "$$repo/scripts/filebase-ipfs-upload.sh" $$files

upload-build-all-gz: build-all
	@$(MAKE) upload-build-all

clean:
	rm -f $(BINARY)

clean-all:
	rm -f $(ARMS_DIR)/bin/arms_*
	rm -f $(ARMS_DIR)/bin/arms_*gz
