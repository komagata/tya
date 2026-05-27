#!/bin/sh
set -eu

cd "$(dirname "$0")/.."

go test ./... -short -count=1
