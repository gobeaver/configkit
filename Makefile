# configkit — developer Makefile
#
# Tooling versions are pinned via `go run` so contributors don't need to
# install anything globally. Bump these deliberately.
STATICCHECK_VERSION := 2024.1.1
GOVULNCHECK_VERSION := latest
GOSEC_VERSION       := v2.21.4

GO            ?= go
PKGS          := ./...
MODULE        := $(shell $(GO) list -m)

# Use `make` with no args to print this list.
.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# ─── Test & quality ─────────────────────────────────────────────────────────

.PHONY: test
test: ## Run tests
	$(GO) test $(PKGS)

.PHONY: test-race
test-race: ## Run tests with the race detector
	$(GO) test -race $(PKGS)

.PHONY: cover
cover: ## Run tests with coverage report
	$(GO) test -coverprofile=coverage.out $(PKGS)
	$(GO) tool cover -func=coverage.out | tail -1

.PHONY: vet
vet: ## go vet
	$(GO) vet $(PKGS)

.PHONY: fmt
fmt: ## Format code
	$(GO) fmt $(PKGS)

.PHONY: tidy
tidy: ## Tidy go.mod
	$(GO) mod tidy

.PHONY: lint
lint: ## Run staticcheck
	$(GO) run honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION) $(PKGS)

.PHONY: vuln
vuln: ## Run govulncheck (Go vulnerability database)
	$(GO) run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) $(PKGS)

.PHONY: gosec
gosec: ## Run gosec (security static analysis)
	$(GO) run github.com/securego/gosec/v2/cmd/gosec@$(GOSEC_VERSION) -quiet $(PKGS)

.PHONY: sec
sec: vuln gosec ## Run all security checks (vuln + gosec)

.PHONY: check
check: fmt vet lint test-race sec ## Run the full pre-release gate

# ─── Release ────────────────────────────────────────────────────────────────

.PHONY: release
release: ## Interactive: tag a new release and push it
	@set -e; \
	if [ -n "$$(git status --porcelain)" ]; then \
		echo "✗ working tree is dirty — commit or stash first"; exit 1; \
	fi; \
	branch=$$(git rev-parse --abbrev-ref HEAD); \
	if [ "$$branch" != "main" ]; then \
		echo "✗ not on main (currently on '$$branch')"; exit 1; \
	fi; \
	last=$$(git describe --tags --abbrev=0 2>/dev/null || echo "(none)"); \
	echo "Module:       $(MODULE)"; \
	echo "Last tag:     $$last"; \
	echo ""; \
	printf "New version (e.g. v0.1.0): "; read new; \
	case "$$new" in \
		v[0-9]*.[0-9]*.[0-9]*) ;; \
		*) echo "✗ '$$new' is not a valid semver tag (must look like vX.Y.Z)"; exit 1 ;; \
	esac; \
	if git rev-parse "$$new" >/dev/null 2>&1; then \
		echo "✗ tag $$new already exists"; exit 1; \
	fi; \
	printf "Release message: "; read msg; \
	if [ -z "$$msg" ]; then echo "✗ message is required"; exit 1; fi; \
	echo ""; \
	echo "About to run pre-release checks, then:"; \
	echo "  git tag -a $$new -m \"$$msg\""; \
	echo "  git push origin $$new"; \
	printf "Proceed? [y/N] "; read confirm; \
	case "$$confirm" in y|Y|yes) ;; *) echo "aborted"; exit 1 ;; esac; \
	$(MAKE) check; \
	git tag -a "$$new" -m "$$msg"; \
	git push origin "$$new"; \
	echo ""; \
	echo "✓ released $$new"; \
	echo "  the Go module proxy will pick it up within a minute:"; \
	echo "  go get $(MODULE)@$$new"

# ─── Housekeeping ───────────────────────────────────────────────────────────

.PHONY: clean
clean: ## Remove build artifacts
	rm -f coverage.out
