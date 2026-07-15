#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: scripts/check-public-release.sh vX.Y.Z" >&2
  exit 1
fi

version="$1"
if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$ ]]; then
  echo "error: version must look like vX.Y.Z" >&2
  exit 1
fi

root="$(git rev-parse --show-toplevel)"
manifest="$root/scripts/module-manifest.txt"
gocache="${GOCACHE:-/tmp/gocache}"
goproxy="${MAIL_RELEASE_GOPROXY:-https://proxy.golang.org}"
if [[ "$goproxy" =~ (^|[,|])direct($|[,|]) ]]; then
  echo "error: MAIL_RELEASE_GOPROXY must not include a direct VCS fallback" >&2
  exit 1
fi
temp_root="$(mktemp -d "${TMPDIR:-/tmp}/mail-public-release.XXXXXX")"
gomodcache="$temp_root/modcache"
cleanup() {
  chmod -R u+w "$temp_root" 2>/dev/null || true
  rm -rf "$temp_root"
}
trap cleanup EXIT

read -r -a test_flags <<< "${GO_TEST_FLAGS:--race -count=1}"

while read -r classification module_dir _; do
  if [[ "$classification" != "published" ]]; then
    continue
  fi

  if [[ "$module_dir" == "." ]]; then
    module_root="$root"
    consumer_name="root"
  else
    module_root="$root/$module_dir"
    consumer_name="${module_dir//\//-}"
  fi

  module_path="$(awk '$1 == "module" { print $2; exit }' "$module_root/go.mod")"
  consumer_dir="$temp_root/$consumer_name"
  mkdir -p "$consumer_dir"

  echo "==> public release $module_path@$version"
  (
    cd "$consumer_dir"
    GOWORK=off GOCACHE="$gocache" GOMODCACHE="$gomodcache" GOPROXY="$goproxy" GONOPROXY=none \
      go mod init "example.com/mail-release-check/$consumer_name" >/dev/null
    GOWORK=off GOCACHE="$gocache" GOMODCACHE="$gomodcache" GOPROXY="$goproxy" GONOPROXY=none \
      go mod download "$module_path@$version"
    GOWORK=off GOCACHE="$gocache" GOMODCACHE="$gomodcache" GOPROXY="$goproxy" GONOPROXY=none \
      go get "$module_path@$version"
    resolved="$(GOWORK=off GOCACHE="$gocache" GOMODCACHE="$gomodcache" GOPROXY="$goproxy" GONOPROXY=none \
      go list -m -f '{{.Version}}' "$module_path")"
    if [[ "$resolved" != "$version" ]]; then
      echo "error: resolved $module_path@$resolved, expected $version" >&2
      exit 1
    fi
    published_dir="$(GOWORK=off GOCACHE="$gocache" GOMODCACHE="$gomodcache" GOPROXY="$goproxy" GONOPROXY=none \
      go list -m -f '{{.Dir}}' "$module_path")"
    if [[ "$published_dir" != "$gomodcache/"* ]]; then
      echo "error: $module_path resolved outside the fresh module cache: $published_dir" >&2
      exit 1
    fi
    GOWORK=off GOCACHE="$gocache" GOMODCACHE="$gomodcache" GOPROXY="$goproxy" GONOPROXY=none \
      go test "${test_flags[@]}" "$module_path/..."
  )
done < "$manifest"
