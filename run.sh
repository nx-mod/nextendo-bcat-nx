#!/usr/bin/env sh
# nextendo-bcat-nx — home-lab launch. Runs straight from the repo: the signing
# key (bcat_signing_key.pem) and a content/ dir ship on this testing branch, so
# no setup is needed. TLS is terminated by the sni-router in front; this serves
# plain HTTP. Override any VAR=value before ./run.sh.
set -e
export BCAT_PORT="${BCAT_PORT:-8470}"
export DASH_PORT="${DASH_PORT:-8110}"
export BCAT_KEY_FILE="${BCAT_KEY_FILE:-bcat_signing_key.pem}"
export BCAT_CONTENT="${BCAT_CONTENT:-content}"
mkdir -p "$BCAT_CONTENT"
echo "[bcat] :$BCAT_PORT  dash :$DASH_PORT  key $BCAT_KEY_FILE  content $BCAT_CONTENT"
exec go run .
