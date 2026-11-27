GO ?= go

BINARY  ?= zuelose
PREFIX  ?= /usr/local
BINDIR  ?= $(PREFIX)/bin
DESTDIR ?=
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

VERSION_PACKAGE = github.com/daum3ns/zuelose/internal/version
LDFLAGS = -s -w -X $(VERSION_PACKAGE).Version=$(VERSION)

.PHONY: build run test check vulncheck install uninstall clean

build:
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o "$(BINARY)" ./cmd/zuelose

run:
	$(GO) run ./cmd/zuelose

test:
	$(GO) test ./...
	python3 -m unittest scripts/test_changelog_release_notes.py

check: build test
	$(GO) vet ./...

vulncheck:
	govulncheck ./...

install: build
	install -d "$(DESTDIR)$(BINDIR)"
	install -m 0755 "$(BINARY)" "$(DESTDIR)$(BINDIR)/zuelose"

uninstall:
	rm -f "$(DESTDIR)$(BINDIR)/zuelose"

clean:
	rm -f "$(BINARY)"
