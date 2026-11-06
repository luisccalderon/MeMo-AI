BINARY  := memo-ai
PREFIX  ?= $(HOME)/.local
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo 0.1.0-dev)
GO      ?= go
LDFLAGS := -s -w -X github.com/luisccalderon/MeMo-AI/internal/version.Version=$(VERSION)
PKG     := ./cmd/memo-ai

.PHONY: all build test vet fmt fmt-check check install uninstall clean

all: check build

build: bin/$(BINARY)

bin/$(BINARY): go.mod $(shell find cmd internal -name '*.go' 2>/dev/null)
	@mkdir -p bin
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(BINARY) $(PKG)

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

fmt-check:
	@out=$$(gofmt -l $$(find . -name '*.go' -not -path './vendor/*')); \
	if [ -n "$$out" ]; then echo "gofmt needed on:"; echo "$$out"; exit 1; fi

check: fmt-check vet test

install: build
	install -d $(DESTDIR)$(PREFIX)/bin
	install -m 0755 bin/$(BINARY) $(DESTDIR)$(PREFIX)/bin/$(BINARY)
	@echo "Installed $(DESTDIR)$(PREFIX)/bin/$(BINARY)"

uninstall:
	rm -f $(DESTDIR)$(PREFIX)/bin/$(BINARY)

clean:
	rm -rf bin dist
