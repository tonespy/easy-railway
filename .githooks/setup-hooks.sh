#!/bin/bash
set -euo pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)
HOOKS_DIR="$REPO_ROOT/.githooks"
TARGET_DIR="$REPO_ROOT/.git/hooks"

echo "Setting up git hooks"

if [ ! -d "$HOOKS_DIR" ]; then
  echo "$HOOKS_DIR directory does not exist"
  exit 1
fi

for hook_file in "$HOOKS_DIR"/*; do
  hook_name=$(basename "$hook_file")

  if [ "$hook_name" = "setup-hooks.sh" ]; then
    continue
  fi

  echo "Installing $hook_name"
  cp -f "$hook_file" "$TARGET_DIR/$hook_name"
  chmod +x "$TARGET_DIR/$hook_name"
done

echo "Git hooks installed"
