SHELL := /bin/sh

APP_ROOT := $(CURDIR)
DATA_ROOT ?= $(abspath $(APP_ROOT)/../Data)
PROJECTS_ROOT ?= $(HOME)/Projects
REGISTRY_DIR := $(DATA_ROOT)/_registry
WORK_DIR := $(DATA_ROOT)/Work
VIEW_DIR := _backoffice/view
SERVER_DIR := _backoffice/server
GO_CACHE ?= /tmp/core-eggs-gocache
PY_CACHE ?= /tmp/core-eggs-pycache
ADDR ?= 127.0.0.1:8787
SESSION_TIMEOUT ?= 10m

.PHONY: help scan workspaces rebuild-index build test check serve serve-live version clean

help:
	@printf '%s\n' 'Core service entry points:'
	@printf '%s\n' ''
	@printf '%s\n' '  make scan        Scan $(PROJECTS_ROOT) and refresh Data/_registry'
	@printf '%s\n' '  make workspaces  Generate Data/Work/* project cards from _registry'
	@printf '%s\n' '  make rebuild-index Rebuild Data/Work/INDEX.md from Work source files'
	@printf '%s\n' '  make build       Build the Svelte backoffice'
	@printf '%s\n' '  make test        Compile scripts and run Go tests'
	@printf '%s\n' '  make check       Run build and tests'
	@printf '%s\n' '  make serve       Serve built backoffice at http://$(ADDR)/'
	@printf '%s\n' '  make version     Print Core runtime version'
	@printf '%s\n' ''
	@printf '%s\n' 'Variables:'
	@printf '%s\n' '  DATA_ROOT=/path      Override the Data tree (default: ../Data next to App)'
	@printf '%s\n' '  PROJECTS_ROOT=/path  Override scan root'
	@printf '%s\n' '  ADDR=host:port       Override serve address'
	@printf '%s\n' '  SESSION_TIMEOUT=10m  Override live agent session timeout'

scan:
	python3 _backoffice/scripts/sniff_projects.py --root "$(PROJECTS_ROOT)" --registry-dir "$(REGISTRY_DIR)"

workspaces:
	python3 _backoffice/scripts/generate_workspaces.py --registry-dir "$(REGISTRY_DIR)" --work-dir "$(WORK_DIR)"

rebuild-index:
	cd "$(SERVER_DIR)" && GOCACHE="$(GO_CACHE)" go run ./cmd/core rebuild-index --root "$(DATA_ROOT)"

build:
	cd "$(VIEW_DIR)" && npm run build

test:
	PYTHONPYCACHEPREFIX="$(PY_CACHE)" python3 -m py_compile _backoffice/scripts/sniff_projects.py _backoffice/scripts/generate_workspaces.py
	cd "$(SERVER_DIR)" && GOCACHE="$(GO_CACHE)" go test ./...

check: build test

serve: build
	if [ -f "$(DATA_ROOT)/.env" ]; then set -a; . "$(DATA_ROOT)/.env"; set +a; fi; \
	cd "$(SERVER_DIR)" && GOCACHE="$(GO_CACHE)" go run ./cmd/core serve --root "$(DATA_ROOT)" --backoffice-dir "$(APP_ROOT)/$(VIEW_DIR)/dist" --addr "$(ADDR)" --session-timeout "$(SESSION_TIMEOUT)"

serve-live:
	@printf '%s\n' 'make serve-live is deprecated; use make serve. Running make serve now.'
	$(MAKE) serve ADDR="$(ADDR)" SESSION_TIMEOUT="$(SESSION_TIMEOUT)"

version:
	cd "$(SERVER_DIR)" && GOCACHE="$(GO_CACHE)" go run ./cmd/core version

clean:
	rm -rf "$(VIEW_DIR)/dist"
