#!/bin/bash
set -e

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT_DIR"

if [ ! -x cmd/server/server ]; then
  echo "[ERROR] cmd/server/server not found. Run quickstart.sh first."
  exit 1
fi
if [ ! -x cmd/timeline-consumer/timeline-consumer ]; then
  echo "[ERROR] cmd/timeline-consumer/timeline-consumer not found. Run quickstart.sh first."
  exit 1
fi

./cmd/server/server &
./cmd/timeline-consumer/timeline-consumer &

echo "[OK] Server and timeline-consumer started."
echo "Press Ctrl+C to stop."
wait
