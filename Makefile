# Toolchain: prefer a mise-managed Go when the one in PATH is not usable
# (`go` and `GOROOT` can disagree on this machine). Override with `make GO=/path/go`.
# When GOROOT points at a different Go than the binary: `GOROOT=/usr/lib/go make check GO=/usr/bin/go`.
SHELL := /bin/bash

GO ?= $(shell ls -d $(HOME)/.local/share/mise/installs/go/*/bin/go 2>/dev/null | tail -1)
ifeq ($(GO),)
GO := go
endif

BIN := bin/wsdash
PKG := ./...

# Same version gitdash and dbx use; run with `go run`, no global binary.
GOLANGCI_LINT_VERSION := v2.13.2
MUTATE_BASE ?= main

# Coverage: the deduplicated `go test -coverprofile` output plus main()'s subprocess profile.
COVER_PROFILE ?= coverage.out
COVER_MAIN ?= .covmain/main.txt

.PHONY: build install test lint fmt-check check run coverage coverage-check mutate-all mutate-all-diff mutate-dry clean

build:
	@mkdir -p bin
	$(GO) build -o $(BIN) ./cmd/wsdash

install: build
	@mkdir -p $(HOME)/.local/bin
	install -m 0755 $(BIN) $(HOME)/.local/bin/wsdash

fmt-check:
	@files="$$(find . -name '*.go' -not -path './bin/*')"; \
	unformatted="$$(gofmt -l $$files)"; \
	if [ -n "$$unformatted" ]; then echo "not gofmt-clean:"; echo "$$unformatted"; exit 1; fi

# lint: fmt-check + golangci-lint (version pinned, always via go run).
lint: fmt-check
	$(GO) run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run

test:
	$(GO) test -race -count=1 $(PKG)

check: build lint test

# .covmain is deleted, not reused: a previous build's counters point at ranges
# today's code no longer has.
coverage:
	@$(GO) test -count=1 -coverpkg ./... -coverprofile=$(COVER_PROFILE) ./... > /dev/null
	@rm -rf .covmain
	@mkdir -p .covmain
	@$(GO) build -cover -o .covmain/wsdash ./cmd/wsdash
	@cfg="$$(mktemp -d)"; mkdir -p "$$cfg/wsdash"; \
		printf 'default_backend = "fake"\n' > "$$cfg/wsdash/config.toml"; \
		XDG_CONFIG_HOME="$$cfg" GOCOVERDIR="$$PWD/.covmain" \
			./.covmain/wsdash --list --no-scan >/dev/null 2>&1 || true; \
		rm -rf "$$cfg"
	@$(GO) tool covdata textfmt -i=.covmain -o=$(COVER_MAIN)
	@echo "profiles: $(COVER_PROFILE) $(COVER_MAIN)"

coverage-check: coverage
	@extra=""; [ -f "$(COVER_MAIN)" ] && extra="-a $$PWD/$(COVER_MAIN)"; \
		scripts/diff-coverage.sh "$(COVER_PROFILE)" "$(MUTATE_BASE)" $$extra

run:
	$(GO) run ./cmd/wsdash

mutate-all:
	@scripts/mutate.sh --run

mutate-all-diff:
	@scripts/mutate.sh --diff

mutate-dry:
	@scripts/mutate.sh --diff --dry

clean:
	rm -rf bin
