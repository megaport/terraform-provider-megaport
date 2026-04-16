#!/usr/bin/env bash
# Prints staging locations with available MVE and port capacity.
#
# Usage:
#   MEGAPORT_ACCESS_KEY=xxx MEGAPORT_SECRET_KEY=xxx ./scripts/find-test-locations.sh
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

: "${MEGAPORT_ACCESS_KEY:?MEGAPORT_ACCESS_KEY must be set}"
: "${MEGAPORT_SECRET_KEY:?MEGAPORT_SECRET_KEY must be set}"
export TF_ACC=1

echo "=== MVE capacity ==="
go test -v -count=1 -run TestListMVECapacity ./internal/provider/

echo ""
echo "=== Port/MCR capacity ==="
go test -v -count=1 -run TestListPortCapacity ./internal/provider/
