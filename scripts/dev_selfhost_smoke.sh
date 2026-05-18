#!/bin/sh
set -eu

cd "$(dirname "$0")/.."

go test ./tests -run 'TestSelfhostV02Scripts/fixed_point' -count=1 -timeout=20m
