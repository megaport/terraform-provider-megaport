#!/usr/bin/env bash
# Lists orphaned tf-acc-test-* resources in staging.
# VXCs are listed/deleted first to free up their A/B-end endpoints.
#
# Usage:
#   # List only (safe):
#   MEGAPORT_ACCESS_KEY=xxx MEGAPORT_SECRET_KEY=xxx ./scripts/cleanup-test-resources.sh
#
#   # Delete orphaned resources:
#   MEGAPORT_ACCESS_KEY=xxx MEGAPORT_SECRET_KEY=xxx ./scripts/cleanup-test-resources.sh --delete
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

: "${MEGAPORT_ACCESS_KEY:?MEGAPORT_ACCESS_KEY must be set}"
: "${MEGAPORT_SECRET_KEY:?MEGAPORT_SECRET_KEY must be set}"
export TF_ACC=1

DELETE_FLAG=""
if [[ "${1:-}" == "--delete" ]]; then
  DELETE_FLAG="-cleanup-delete"
  echo "WARNING: --delete specified. Orphaned resources will be deleted."
  echo "Press Ctrl-C within 5 seconds to abort..."
  sleep 5
fi

go test -v -count=1 -run TestCleanupOrphanedResources ./internal/provider/ ${DELETE_FLAG}
