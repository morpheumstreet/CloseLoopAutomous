#!/usr/bin/env bash
# Build fishtank and sync static output to a deploy directory.
# If the destination is a git repo, runs git add/commit/push there.
# Usage: bash build-and-deploy.sh [destination_directory]
# Env:   DEPLOY overrides default destination when $1 is omitted.
#        GIT_COMMIT_MSG overrides the default deploy commit message.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEFAULT_DEPLOY="/Users/hesdx/Documents/toolings/deploy-missioncontrol"


if [[ ! -f "$ROOT/package.json" ]]; then
  echo "error: package.json not found in $ROOT" >&2
  exit 1
fi

cd "$ROOT"
echo "Running bun run build..."
bun run build

if [[ ! -d "$ROOT/dist" ]]; then
  echo "error: dist directory was not created after build" >&2
  exit 1
fi

if [[ -f "$ROOT/vercel.json" ]]; then
  cp "$ROOT/vercel.json" "$ROOT/dist/vercel.json"
  echo "Copied vercel.json into dist/"
fi

# mkdir -p "$DEFAULT_DEPLOY"
echo "Syncing dist/ to $DEFAULT_DEPLOY ..."
# --delete removes extra files in DEST; protect /.git/ so a repo at $DEFAULT_DEPLOY is not wiped.
rsync -a --delete --filter='protect /.git/' "$ROOT/dist/" "$DEFAULT_DEPLOY/"
echo "Build completed. Deployed to $DEFAULT_DEPLOY"

if git -C "$DEFAULT_DEPLOY" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "Staging, committing, and pushing deploy repo at $DEFAULT_DEPLOY ..."
  git -C "$DEFAULT_DEPLOY" add -A
  if git -C "$DEFAULT_DEPLOY" diff --cached --quiet; then
    echo "No changes to commit in deploy repo."
  else
    MSG="${GIT_COMMIT_MSG:-Deploy $(date -u +%Y-%m-%dT%H:%M:%SZ)}"
    git -C "$DEFAULT_DEPLOY" commit -m "$MSG"
    git -C "$DEFAULT_DEPLOY" push
    echo "Pushed deploy repo."
  fi
else
  echo "Note: $DEFAULT_DEPLOY is not a git repository; skipping git add/commit/push."
fi
