#!/usr/bin/env bash
#
# generate-changelog.sh: produces CHANGELOG.md from git tags and a curated V2 header.
# Invoked by: go generate ./...
#
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="${REPO_ROOT}/CHANGELOG.md"

# Shallow clones (for example CI checkouts without fetch-depth: 0, or local
# --depth clones) may have some tags but not the full commit history between
# them. Generating from that state would silently write truncated per-release
# commit lists, so skip regeneration entirely rather than overwrite CHANGELOG.md
# with incomplete data.
if [ "$(git -C "$REPO_ROOT" rev-parse --is-shallow-repository)" = "true" ]; then
    echo "Repository is a shallow clone; leaving existing CHANGELOG.md unchanged." >&2
    echo "Fetch full history (for example, 'git fetch --unshallow') to regenerate it." >&2
    exit 0
fi

TMPOUT="$(mktemp "${OUT}.XXXXXX")"
trap 'rm -f "$TMPOUT"' EXIT

cat > "$TMPOUT" << 'HEADER'
# Changelog

All notable changes to the Megaport Terraform Provider will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased] - V2

### Breaking Changes

- Module path changed to `github.com/megaport/terraform-provider-megaport/v2`

HEADER

# Append release history generated from git tags
echo "---" >> "$TMPOUT"
echo "" >> "$TMPOUT"
echo "## Release History" >> "$TMPOUT"
echo "" >> "$TMPOUT"

# Restrict to tags merged into the current HEAD so changelog generation only
# considers releases reachable from this checkout/branch.
# Exclude prerelease tags (alpha/beta/rc): their commits are covered by the
# final release entry and Git's version sort does not follow SemVer prerelease
# ordering. Tolerate zero matching tags (grep returns 1 on no match).
tags=()
while IFS= read -r t; do
    [[ -n "$t" ]] && tags+=("$t")
done < <(git -C "$REPO_ROOT" tag --merged HEAD --list 'v*' --sort=-version:refname | grep -Ev '\-(alpha|beta|rc)' || true)

if [ "${#tags[@]}" -eq 0 ]; then
    echo "No matching release tags found; writing curated header with empty release history." >&2
    echo "Ensure tags are fetched (for example, in shallow clones or CI environments) to include one." >&2
    echo "_No release tags found in this checkout._" >> "$TMPOUT"
    echo "" >> "$TMPOUT"
    mv "$TMPOUT" "$OUT"
    trap - EXIT
    exit 0
fi

for i in "${!tags[@]}"; do
    tag="${tags[$i]}"
    date=$(git -C "$REPO_ROOT" log -1 --format="%as" "$tag")
    echo "### [$tag] - $date" >> "$TMPOUT"
    echo "" >> "$TMPOUT"

    # Range: commits in this tag that are NOT in the next-older tag
    next_idx=$((i + 1))
    if [ "$next_idx" -lt "${#tags[@]}" ]; then
        prev_tag="${tags[$next_idx]}"
        range="${prev_tag}..${tag}"
    else
        # Oldest tag: show its commits only
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
            | sed 's/—/-/g' \
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
