#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

OUTPUT_FILE="${COVERAGE_OUTPUT:-coverage.txt}"
if [[ -n "${COVERAGE_TMP_DIR:-}" ]]; then
  mkdir -p "$COVERAGE_TMP_DIR"
  TMP_ROOT="$(mktemp -d "$COVERAGE_TMP_DIR/mail-coverage.XXXXXX")"
else
  TMP_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/mail-coverage.XXXXXX")"
fi
GOCACHE_DIR="${GOCACHE_DIR:-${GOCACHE:-/tmp/gocache}}"
GOMODCACHE_DIR="${GOMODCACHE_DIR:-${GOMODCACHE:-/tmp/gomodcache}}"

cleanup() {
  rm -rf "$TMP_ROOT"
}
trap cleanup EXIT

ROOT_COVER_DIR="$TMP_ROOT/root"
SES_COVER_DIR="$TMP_ROOT/mailses"
MERGED_DIR="$TMP_ROOT/merged"

mkdir -p "$ROOT_COVER_DIR" "$SES_COVER_DIR" "$MERGED_DIR" "$GOCACHE_DIR" "$GOMODCACHE_DIR"

echo "==> Root module coverage"
GOCACHE="$GOCACHE_DIR" GOMODCACHE="$GOMODCACHE_DIR" \
go test -cover ./... -args -test.gocoverdir="$ROOT_COVER_DIR"

if [[ -d "mailses" && -f "mailses/go.mod" ]]; then
  echo "==> mailses module coverage"
  (
    cd mailses
    GOCACHE="$GOCACHE_DIR" GOMODCACHE="$GOMODCACHE_DIR" \
    go test -cover ./... -args -test.gocoverdir="$SES_COVER_DIR"
  )
fi

echo "==> Merge coverage"
go tool covdata merge -i="$ROOT_COVER_DIR,$SES_COVER_DIR" -o="$MERGED_DIR"

mkdir -p "$(dirname "$OUTPUT_FILE")"
go tool covdata textfmt -i="$MERGED_DIR" -o="$OUTPUT_FILE"

echo "==> Combined coverage written to $OUTPUT_FILE"
go tool covdata percent -i="$MERGED_DIR"
awk -F'[: ]+' 'NR>1 {covered += ($4 > 0 ? $3 : 0); total += $3} END {printf "==> Combined total %.1f%%\n", (covered/total)*100}' "$OUTPUT_FILE"
