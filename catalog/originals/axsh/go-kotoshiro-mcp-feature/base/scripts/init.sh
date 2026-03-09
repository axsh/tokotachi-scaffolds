#!/usr/bin/env bash
set -e
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
# shellcheck source=scripts/_common.sh
. "$SCRIPT_DIR/_common.sh"

cd "$FUNCTION_ROOT"
docker build --build-arg FUNCTION_NAME="$FUNCTION_NAME" -t "$IMAGE_DEV" -f container/dev/Dockerfile .
# Populate the shared go module cache once so run.sh/build.sh do not run go mod every time.
docker run --rm \
  -v "$FUNCTION_ROOT:/workspace/$FUNCTION_NAME" \
  -v "gomodcache_${FUNCTION_NAME}:/go/pkg/mod" \
  -e GOMODCACHE=/go/pkg/mod \
  -w "/workspace/$FUNCTION_NAME" \
  "$IMAGE_DEV" \
  go mod download
