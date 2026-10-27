.DEFAULT_GOAL := ci

GO ?= go
BINARY ?= b64split

.PHONY: ci fmt-check vet test build

ci: fmt-check vet test build

fmt-check:
	@unformatted=$$(find . -type d \( -name .git -o -name vendor \) -prune -o -type f -name '*.go' -exec gofmt -l {} +); \
	if [ -n "$$unformatted" ]; then \
		printf 'Go files need gofmt:\n%s\n' "$$unformatted"; \
		exit 1; \
	fi

vet:
	$(GO) vet ./...

test:
	$(GO) test ./...

build:
	$(GO) build -trimpath -o $(BINARY) .
