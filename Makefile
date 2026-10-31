# Toolchain: prefer a mise-managed Go when the one in PATH is not usable
# (`go` and `GOROOT` can disagree on this machine). Override with `make GO=/path/go`.
GO ?= $(shell ls -d $(HOME)/.local/share/mise/installs/go/*/bin/go 2>/dev/null | tail -1)
ifeq ($(GO),)
GO := go
endif

BIN := bin/wsdash
PKG := ./...

.PHONY: build install test lint fmt-check check run clean

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

lint: fmt-check
	$(GO) vet $(PKG)

test:
	$(GO) test -race -count=1 $(PKG)

check: build lint test

run:
	$(GO) run ./cmd/wsdash

clean:
	rm -rf bin
