#!/usr/bin/env bash
set -e

echo "Running all Go tests (verbose)"
go test -v ./...

echo "All tests finished"
