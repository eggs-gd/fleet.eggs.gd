SHELL := /bin/sh

APP_ROOT := $(CURDIR)

# Machine-local overrides (not committed): DATA_ROOT, ADDR, LIVE, PROJECTS_ROOTS.
# Included after APP_ROOT so a DATA_ROOT that references it (e.g.
# $(abspath $(APP_ROOT)/../Data)) resolves against this checkout, not "".
-include Makefile.local

DATA_ROOT ?=
VIEW_DIR := view
SITE_DIR := site
SERVER_DIR := server
WEBUI_DIST := $(SERVER_DIR)/internal/webui/dist
BIN := $(APP_ROOT)/bin/fleet
ADDR ?= 127.0.0.1:8787
SESSION_TIMEOUT ?= 10m
LIVE ?=
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null | sed 's/^v//')
LDFLAGS := $(if $(VERSION),-X main.version=$(VERSION),)

.PHONY: help setup ui build test vet check serve scan rebuild-index version fmt site clean

help:
	@printf '%s\n' 'Fleet entry points:'
	@printf '%s\n' ''
	@printf '%s\n' '  make setup         Install dashboard and site dependencies (npm ci)'
	@printf '%s\n' '  make build         Build the dashboard, then bin/fleet with the dashboard inside'
	@printf '%s\n' '  make serve         Build, then run bin/fleet serve on http://$(ADDR)/'
	@printf '%s\n' '  make test          Run the Go tests'
	@printf '%s\n' '  make vet           Run go vet'
	@printf '%s\n' '  make check         Run vet, the Go tests, and the dashboard tests'
	@printf '%s\n' '  make scan          Scan PROJECTS_ROOTS and refresh the registry'
	@printf '%s\n' '  make rebuild-index Rebuild Work/INDEX.md from the task files'
	@printf '%s\n' '  make site          Typecheck and build the static landing'
	@printf '%s\n' '  make fmt           gofmt for the server, prettier for view and site'
	@printf '%s\n' '  make clean         Remove build output'
	@printf '%s\n' ''
	@printf '%s\n' 'Variables (or put them in Makefile.local):'
	@printf '%s\n' '  DATA_ROOT=/path      Data root. Default: the one saved in ~/.fleet/app.json,'
	@printf '%s\n' '                       created at ~/.fleet/workspace on first run'
	@printf '%s\n' '  LIVE=1               Start agents for ready tasks (default: plan launches only)'
	@printf '%s\n' '  ADDR=host:port       Listen address'
	@printf '%s\n' '  SESSION_TIMEOUT=10m  Idle attention threshold for agent sessions'
	@printf '%s\n' '  PROJECTS_ROOTS="a b" Directories to scan (make scan)'

setup:
	cd "$(VIEW_DIR)" && npm ci
	cd "$(SITE_DIR)" && npm ci

# The dashboard is copied into the Go package that embeds it. The placeholder
# stays, so a checkout that has not built the dashboard still compiles.
ui:
	cd "$(VIEW_DIR)" && npm run build
	find "$(WEBUI_DIST)" -mindepth 1 ! -name PLACEHOLDER -delete
	cp -R "$(VIEW_DIR)/dist/." "$(WEBUI_DIST)/"

build: ui
	cd "$(SERVER_DIR)" && go build -ldflags '$(LDFLAGS)' -o "$(BIN)" ./cmd/fleet

test:
	cd "$(SERVER_DIR)" && go test ./...

vet:
	cd "$(SERVER_DIR)" && go vet ./...

check: vet test
	cd "$(VIEW_DIR)" && npm run check && npm run test:ui-state && npm run test:technology-display && npm run format:check

serve: build
	$(if $(DATA_ROOT),if [ -f "$(DATA_ROOT)/.env" ]; then set -a; . "$(DATA_ROOT)/.env"; set +a; fi;) \
	"$(BIN)" serve --addr "$(ADDR)" --session-timeout "$(SESSION_TIMEOUT)" $(if $(DATA_ROOT),--root "$(DATA_ROOT)") $(if $(LIVE),--live)

scan: build
	@test -n "$(PROJECTS_ROOTS)" || { printf '%s\n' 'PROJECTS_ROOTS is required'; exit 1; }
	"$(BIN)" scan $(if $(DATA_ROOT),--root "$(DATA_ROOT)") $(foreach root,$(PROJECTS_ROOTS),--projects "$(root)")

rebuild-index: build
	"$(BIN)" rebuild-index $(if $(DATA_ROOT),--root "$(DATA_ROOT)")

version: build
	"$(BIN)" version

site:
	cd "$(SITE_DIR)" && npm run check && npm run build

fmt:
	cd "$(SERVER_DIR)" && gofmt -w .
	cd "$(VIEW_DIR)" && npm run format
	cd "$(SITE_DIR)" && npm run format

clean:
	rm -rf "$(VIEW_DIR)/dist" "$(APP_ROOT)/bin"
	find "$(WEBUI_DIST)" -mindepth 1 ! -name PLACEHOLDER -delete
