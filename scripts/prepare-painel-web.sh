#!/usr/bin/env bash
# Gera painel-web-react/dist e copia para servidor-go/assets (para commit ou upload no deploy).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
REPO="$(cd "$ROOT/.." && pwd)"
cd "$REPO/painel-web-react"
if [[ ! -f package.json ]]; then
  echo "pasta painel-web-react nao encontrada em $REPO/painel-web-react" >&2
  exit 1
fi
npm ci
npm run build:deploy
echo "Pronto: assets em $ROOT/assets"
