#!/usr/bin/env bash
# start.sh — build and run the HOKM platform, then open the game.
# Usage:  ./start.sh          (dev stack: frontend :5173, backend :8080)

set -euo pipefail
cd "$(dirname "$0")"

if ! command -v docker >/dev/null 2>&1; then
  echo "ERROR: docker not found — start Docker Desktop first." >&2
  exit 1
fi

# First run: create .env from the template.
if [ ! -f .env ]; then
  cp .env.example .env
  echo "created .env from .env.example (dev defaults)"
fi

echo "building and starting containers..."
docker compose up --build -d

echo "waiting for the backend to become healthy..."
ok=""
for _ in $(seq 1 60); do
  if curl -sf http://localhost:8080/api/health >/dev/null 2>&1; then ok=yes; break; fi
  sleep 2
done
if [ -z "$ok" ]; then
  echo "WARNING: backend not healthy yet — check: docker compose logs backend" >&2
fi

echo
echo "  HOKM is running:  http://localhost:5173"
echo "  API:              http://localhost:8080/api/health"
echo "  stop:             docker compose down"
echo

# Open the game in the default browser (best effort).
if command -v xdg-open >/dev/null 2>&1; then xdg-open http://localhost:5173
elif command -v open >/dev/null 2>&1; then open http://localhost:5173
elif command -v cmd.exe >/dev/null 2>&1; then cmd.exe /c start http://localhost:5173
fi

docker compose ps
