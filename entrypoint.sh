#!/bin/sh
set -e

# Default to API if SERVICE is not set
SERVICE=${SERVICE:-api}

case "$SERVICE" in
  api)
    exec /app/bin/api
    ;;
  websocket)
    exec /app/bin/websocket
    ;;
  worker)
    exec /app/bin/worker
    ;;
  *)
    echo "Unknown SERVICE: $SERVICE"
    exit 1
    ;;
esac
