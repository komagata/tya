#!/bin/sh
set -eu

cd "$(dirname "$0")/.."

TYA_RUN_LONG_SELFHOST=1 go test ./tests -run 'TestSelfhostV02Scripts|TestBootstrapNoGoSelfhostV02FixedPoint' -count=1 -timeout=20m
go test ./... -count=1 -timeout=20m
