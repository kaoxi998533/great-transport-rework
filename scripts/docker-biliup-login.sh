#!/usr/bin/env bash
set -euo pipefail

IMAGE="${IMAGE:-yt-transfer}"
COOKIE_PATH="${COOKIE_PATH:-$PWD/../cookies.json}"

touch "$COOKIE_PATH"

echo "Using cookies at $COOKIE_PATH"
echo "Launching biliup login inside $IMAGE..."
docker run --rm -it \
  -v "$COOKIE_PATH:/app/cookies.json" \
  --entrypoint /bin/sh \
  "$IMAGE" \
  -c 'python3 -m biliup login --user-cookie /app/cookies.json "$@"' sh "$@"
