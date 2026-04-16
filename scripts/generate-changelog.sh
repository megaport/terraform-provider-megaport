#!/usr/bin/env bash
#
# generate-changelog.sh — produces CHANGELOG.md from git tags and a curated V2 header.
# Invoked by: go generate ./...
#
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="${REPO_ROOT}/CHANGELOG.md"
TMPOUT="$(mktemp "${OUT}.XXXXXX")"
trap 'rm -f "$TMPOUT"' EXIT

cat > "$TMPOUT" << 'HEADER'
# Changelog

All notable changes to the Megaport Terraform Provider will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased] — V2

### Breaking Changes

- Module path changed to `github.com/megaport/terraform-provider-megaport/v2`
- Removed read-only metadata fields from all resources (`product_id`, `provisioning_status`, `usage_algorithm`, `virtual`, `locked`, `cancelable`, `vxc_permitted`, `vxc_auto_approval`, `live_date`, `create_date`, `created_by`, `market`, `terminate_date`, `contract_start_date`, `contract_end_date`)
- VXC partner configs moved inside `a_end_config`/`b_end_config` blocks
- MVE `vendor_config` replaced with per-vendor blocks (`aruba_config`, `cisco_config`, etc.)
- MCR inline `prefix_filter_lists` removed — use `megaport_mcr_prefix_filter_list` resource
- Removed `last_updated` field from all resources
- Removed `ordered_vlan` from VXC — use `vlan` only
- Date formats standardized to RFC 3339
- VXC end config UIDs renamed: `requested_product_uid` → `product_uid`, `current_product_uid` → `assigned_product_uid`

### Added

- `ResourceWithMoveState` for automatic V1 → V2 state migration
- Per-resource configurable timeouts via `timeouts` block
- Data source: `megaport_vxc_csp_connection`
- Shared retry/backoff utilities with exponential backoff and jitter
- Enriched API error messages with HTTP status and trace ID
- Unit tests for `fromAPI` mapping functions across all resources
- `GNUmakefile` with standard build targets

### Fixed

- IX resource now respects provider-level `wait_time` setting
- MCR rate limiter goroutine leak in prefix filter list operations
- Silent diagnostic swallowing in `fromAPI` methods
- Global `waitForTime` variable thread-safety issue (moved to per-resource field)

### Changed

- Retry strategies standardized across all resources (exponential backoff + jitter)
- Shared `configureMegaportResource` helper extracted for all resources
- Port resource helpers extracted into `port_resource_utils.go`

HEADER

# Append release history generated from git tags
echo "---" >> "$TMPOUT"
echo "" >> "$TMPOUT"
echo "## Release History" >> "$TMPOUT"
echo "" >> "$TMPOUT"

# Restrict to tags merged into the current HEAD so changelog generation only
# considers releases reachable from this checkout/branch.
# Exclude prerelease tags (alpha/beta/rc) — their commits are covered by the
# final release entry and Git's version sort does not follow SemVer prerelease
# ordering. Tolerate zero matching tags (grep returns 1 on no match).
tags=()
while IFS= read -r t; do
    [[ -n "$t" ]] && tags+=("$t")
done < <(git -C "$REPO_ROOT" tag --merged HEAD --list 'v*' --sort=-version:refname | grep -Ev '\-(alpha|beta|rc)' || true)

if [ "${#tags[@]}" -eq 0 ]; then
    echo "No matching release tags found; leaving existing CHANGELOG.md unchanged." >&2
    echo "Ensure tags are fetched (for example, in shallow clones or CI environments)." >&2
    exit 1
fi

for i in "${!tags[@]}"; do
    tag="${tags[$i]}"
    date=$(git -C "$REPO_ROOT" log -1 --format="%as" "$tag" 2>/dev/null)
    echo "### [$tag] — $date" >> "$TMPOUT"
    echo "" >> "$TMPOUT"

    # Range: commits in this tag that are NOT in the next-older tag
    next_idx=$((i + 1))
    if [ "$next_idx" -lt "${#tags[@]}" ]; then
        prev_tag="${tags[$next_idx]}"
        range="${prev_tag}..${tag}"
    else
        # Oldest tag — show its commits only
        range="$tag"
    fi

    # Filter out unscoped "chore:" commits but keep scoped ones like "chore(deps):"
    # which carry useful information (dependency bumps, etc.).
    commits=$(git -C "$REPO_ROOT" log --no-merges --format="- %s" "$range") || {
        echo "error: git log failed for range $range" >&2
        exit 1
    }
    filtered=$(
        printf '%s\n' "$commits" \
            | grep -Ev '^- chore:' \
            | sed 's/^- \(#\)/- \\\1/' \
            | sed 's/  */ /g' || true
    )
    if [ -n "$filtered" ]; then
        printf '%s\n' "$filtered" >> "$TMPOUT"
    else
        echo "- _No user-facing changes._" >> "$TMPOUT"
    fi

    echo "" >> "$TMPOUT"
done

mv "$TMPOUT" "$OUT"
trap - EXIT
