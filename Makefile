BINARY := aasdd
SPEC   := spec/spec.md

# Detect Apple Silicon — Docker-in-Docker is unsupported inside act on arm64 macOS
ifeq ($(shell uname -s)-$(shell uname -m),Darwin-arm64)
  ACT_FLAGS := --env SKIP_DOCKER=true
else
  ACT_FLAGS :=
endif

.PHONY: help build run test lint release-build release-test clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*##"}; {printf "  %-18s %s\n", $$1, $$2}'

## ── Build & Run ──────────────────────────────────────────────────────────────

build: ## Build the CLI binary to ./aasdd
	go build -o $(BINARY) .

run: ## Run the CLI (e.g. make run ARGS="verify ./spec")
	go run . $(ARGS)

## ── Test ─────────────────────────────────────────────────────────────────────

test: ## Run all Go tests
	go test ./...

verify: build ## Verify the spec using the locally built CLI
	./$(BINARY) verify ./spec

## ── Release ──────────────────────────────────────────────────────────────────
## Actual releases are triggered by pushing a vX.Y.Z tag; the GitHub Actions
## workflow handles publishing. The targets below are for local validation only.

release-build: ## Build all artifacts locally incl. Docker images (requires goreleaser + buildx)
	goreleaser release --snapshot --clean

release-test: ## Simulate the CI workflow via act; skips Docker on Apple Silicon (use release-build for full Docker coverage)
	act push \
		--secret-file .secrets \
		--eventpath .act/push-tag.json \
		--job release \
		-W .github/workflows/release-local.yml \
		$(ACT_FLAGS)

## ── Housekeeping ─────────────────────────────────────────────────────────────

clean: ## Remove build artifacts and locally built Docker images
	rm -f $(BINARY)
	rm -rf dist/
	docker images --format '{{.Repository}}:{{.Tag}}' | grep '^ghcr.io/smithyai/aasdd-cli:' | xargs -r docker rmi --force
