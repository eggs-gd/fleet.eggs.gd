#!/bin/sh
# Temporary helper from CORE-85 agent session; safe to delete.
cd "$(dirname "$0")" && go build ./... && go vet ./... && go test ./internal/corechain/ -count=1
