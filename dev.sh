#!/usr/bin/env bash
set -euo pipefail

# Runs the full dev stack: vite (frontend HMR) + air (Go live reload).
# Stops both on Ctrl-C.

cleanup() {
  echo
  echo "==> stopping dev stack"
  kill 0 2>/dev/null || true
}
trap cleanup EXIT INT TERM

echo "==> vite (frontend HMR)"
npm run dev &

echo "==> air (Go live reload)"
air &

wait
