.DEFAULT_GOAL := ci
.DELETE_ON_ERROR:

GO ?= go
GOFMT ?= gofmt
BINARY ?= bin/b64split
DIST_DIR ?= dist
VERSION ?= dev
PLATFORMS ?= darwin/amd64 darwin/arm64 linux/amd64 linux/arm64

.PHONY: ci fmt fmt-check vet test build dist release release-check repo-path

ci: fmt-check vet test
	@set -eu; \
	workdir=$$(mktemp -d); \
	trap 'rm -rf "$$workdir"' EXIT HUP INT TERM; \
	$(MAKE) build BINARY="$$workdir/b64split"

fmt:
	$(GOFMT) -w $(wildcard *.go)

fmt-check:
	@set -eu; unformatted=$$($(GOFMT) -l $(wildcard *.go)); \
	if [ -n "$$unformatted" ]; then \
		printf 'Go files need gofmt:\n%s\n' "$$unformatted"; \
		exit 1; \
	fi

vet:
	$(GO) vet ./...

test:
	$(GO) test -race -count=1 ./...

build:
	@mkdir -p "$(dir $(BINARY))"
	$(GO) build -trimpath -o "$(BINARY)" .

dist:
	@set -eu; \
	case "$(VERSION)" in ''|*[!a-zA-Z0-9._-]*) echo 'VERSION must contain only letters, digits, dots, underscores, or hyphens' >&2; exit 1;; esac; \
	mkdir -p "$(DIST_DIR)"; \
	stage=$$(mktemp -d); \
	trap 'rm -rf "$$stage"' EXIT HUP INT TERM; \
	for platform in $(PLATFORMS); do \
		os=$${platform%/*}; arch=$${platform#*/}; \
		name="b64split_$(VERSION)_$${os}_$${arch}"; \
		mkdir -p "$$stage/$$name"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch $(GO) build -trimpath -o "$$stage/$$name/b64split" .; \
		cp README.md "$$stage/$$name/"; \
		tar -czf "$$stage/$$name.tar.gz" -C "$$stage/$$name" b64split README.md; \
	done; \
	(cd "$$stage" && shasum -a 256 ./*.tar.gz > checksums.txt); \
	cp "$$stage/"*.tar.gz "$$stage/checksums.txt" "$(DIST_DIR)/"

release-check:
	@test "$(VERSION)" != dev || { echo 'Set VERSION to an existing release tag, e.g. v1.0.0' >&2; exit 1; }
	@git diff --quiet && git diff --cached --quiet || { echo 'Release requires a clean working tree' >&2; exit 1; }
	@test -z "$$(git ls-files --others --exclude-standard)" || { echo 'Release requires a clean working tree with no untracked files' >&2; exit 1; }
	git rev-parse --verify "refs/tags/$(VERSION)"
	@test "$$(git rev-parse HEAD)" = "$$(git rev-list -n 1 '$(VERSION)')" || { echo 'HEAD must match VERSION' >&2; exit 1; }

release: release-check ci dist
	gh release create "$(VERSION)" $(foreach platform,$(PLATFORMS),"$(DIST_DIR)/b64split_$(VERSION)_$(subst /,_,$(platform)).tar.gz") "$(DIST_DIR)/checksums.txt" --verify-tag --generate-notes

repo-path:
	@pwd
