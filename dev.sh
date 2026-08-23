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

# The Go app reads public/hot once at boot to decide whether to serve vite's
# HMR asset URLs or the production manifest. Clear any stale file from a
# previous crashed run and wait for vite to write a fresh one before starting
# air — otherwise air can win the race and boot in production-asset mode for
# its whole run.
rm -f public/hot

echo "==> vite (frontend HMR)"
npm run dev &

echo "==> waiting for vite dev server..."
until [ -f public/hot ]; do sleep 0.1; done

echo "==> air (Go live reload)"
air &

wait
