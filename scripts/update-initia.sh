#!/usr/bin/env bash
# Update github.com/initia-labs/initia to a target version across all go.mod
# files in this monorepo, and sync replace directives from the upstream initia
# go.mod so they stay in lockstep.
#
# Usage:
#   ./scripts/update-initia.sh <version>
#   ./scripts/update-initia.sh <version> --no-tidy
#
# Examples:
#   ./scripts/update-initia.sh v1.4.2
#   ./scripts/update-initia.sh v1.4.2 --no-tidy

set -euo pipefail

# ── args ─────────────────────────────────────────────────────────────────────

if [ $# -lt 1 ]; then
  echo "Usage: $0 <version> [--no-tidy]"
  echo ""
  echo "Updates github.com/initia-labs/initia to <version> and syncs replace"
  echo "directives from the upstream initia go.mod."
  echo ""
  echo "Examples:"
  echo "  $0 v1.4.2"
  echo "  $0 v1.4.2 --no-tidy"
  exit 1
fi

VERSION="$1"
RUN_TIDY=1
for arg in "$@"; do
  case "$arg" in
    --no-tidy) RUN_TIDY=0 ;;
  esac
done

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
UPSTREAM_URL="https://raw.githubusercontent.com/initia-labs/initia/${VERSION}/go.mod"

# ── fetch upstream go.mod ────────────────────────────────────────────────────

echo "==> Fetching upstream go.mod: $UPSTREAM_URL"

UPSTREAM_GOMOD=$(curl -fsSL "$UPSTREAM_URL") || {
  echo "ERROR: failed to fetch go.mod for $VERSION"
  echo "       make sure the tag exists at github.com/initia-labs/initia"
  exit 1
}

echo "    ok"

# ── parse replace directives from upstream ───────────────────────────────────

extract_replace_target() {
  local module="$1"
  echo "$UPSTREAM_GOMOD" \
    | grep -E "^[[:space:]]+$(echo "$module" | sed 's|/|\\/|g') =>" \
    | head -1 \
    | sed -E 's/.*=> //' \
    | sed -E 's/[[:space:]]*$//'
}

REPLACE_MODULES=(
  "github.com/cometbft/cometbft"
  "github.com/cosmos/cosmos-sdk"
  "github.com/cosmos/iavl"
  "github.com/noble-assets/forwarding/simapp"
  "github.com/noble-assets/forwarding/v2"
  "github.com/skip-mev/connect/v2"
  "github.com/99designs/keyring"
  "github.com/dgrijalva/jwt-go"
  "github.com/gin-gonic/gin"
  "github.com/syndtr/goleveldb"
)

REPLACE_TARGETS=()
for mod in "${REPLACE_MODULES[@]}"; do
  REPLACE_TARGETS+=("$(extract_replace_target "$mod")")
done

INITIA_API_VERSION=$(
  echo "$UPSTREAM_GOMOD" \
    | grep -E 'github\.com/initia-labs/initia/api' \
    | head -1 \
    | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+[^ ]*' \
    | head -1
)

echo ""
echo "==> Parsed from upstream:"
echo "    initia:              $VERSION"
echo "    initia/api:          ${INITIA_API_VERSION:-<same as initia>}"
for i in "${!REPLACE_MODULES[@]}"; do
  target="${REPLACE_TARGETS[$i]}"
  if [ -n "$target" ]; then
    printf "    %-22s => %s\n" "$(basename "${REPLACE_MODULES[$i]}")" "$target"
  fi
done

: "${INITIA_API_VERSION:=$VERSION}"

# ── discover modules ─────────────────────────────────────────────────────────

MOD_DIRS=()
while IFS= read -r modfile; do
  MOD_DIRS+=("$(dirname "$modfile")")
done < <(find "$ROOT_DIR" -name go.mod -not -path '*/vendor/*')

echo ""
echo "==> Found ${#MOD_DIRS[@]} module(s)"

# ── helpers ──────────────────────────────────────────────────────────────────

sed_inplace() {
  local file="$1" expr="$2"
  if [[ "$OSTYPE" == darwin* ]]; then
    sed -i '' -E "$expr" "$file"
  else
    sed -i -E "$expr" "$file"
  fi
}

update_replace() {
  local file="$1" module="$2" target="$3"
  local escaped
  escaped=$(echo "$module" | sed 's|/|\\/|g; s|\.|\\.|g')
  sed_inplace "$file" \
    "s|(${escaped} =>) .*|\1 ${target}|"
}

# ── update each go.mod ───────────────────────────────────────────────────────

MISSING_REPLACE_ERRORS=()

for dir in "${MOD_DIRS[@]}"; do
  f="$dir/go.mod"
  rel="${dir#$ROOT_DIR/}"
  echo ""
  echo "--- $rel/go.mod ---"

  # 1) update require: github.com/initia-labs/initia vX.Y.Z
  #    match "initia " (space) so we don't clobber initia/api, initia/movevm, etc.
  if grep -qE 'github\.com/initia-labs/initia ' "$f"; then
    sed_inplace "$f" \
      "s|(github\.com/initia-labs/initia) v[[:alnum:]._-]+|\1 ${VERSION}|g"
    echo "  initia -> $VERSION"
  fi

  # 2) update require: github.com/initia-labs/initia/api vX.Y.Z
  if grep -qE 'github\.com/initia-labs/initia/api ' "$f"; then
    sed_inplace "$f" \
      "s|(github\.com/initia-labs/initia/api) v[[:alnum:]._-]+|\1 ${INITIA_API_VERSION}|g"
    echo "  initia/api -> $INITIA_API_VERSION"
  fi

  # 3) sync replace directives from upstream; flag missing ones
  for i in "${!REPLACE_MODULES[@]}"; do
    mod="${REPLACE_MODULES[$i]}"
    target="${REPLACE_TARGETS[$i]}"
    [ -z "$target" ] && continue
    escaped_mod=$(echo "$mod" | sed 's|/|\\/|g')
    if grep -qE "${escaped_mod} =>" "$f" 2>/dev/null; then
      update_replace "$f" "$mod" "$target"
      echo "  replace $mod => $target"
    elif grep -qE "[[:space:]]${escaped_mod} " "$f" 2>/dev/null; then
      echo "  MISSING replace $mod => $target"
      MISSING_REPLACE_ERRORS+=("$rel/go.mod: $mod => $target")
    fi
  done
done

if [ ${#MISSING_REPLACE_ERRORS[@]} -gt 0 ]; then
  echo ""
  echo "ERROR: the following go.mod files require modules that upstream initia"
  echo "       replaces, but are missing the corresponding replace directives:"
  for entry in "${MISSING_REPLACE_ERRORS[@]}"; do
    echo "         $entry"
  done
  echo ""
  echo "       Add the missing replace directives and re-run this script."
  exit 1
fi

# ── go mod tidy ──────────────────────────────────────────────────────────────

if [[ "$RUN_TIDY" -eq 1 ]]; then
  echo ""
  echo "==> Running go mod tidy ..."
  for dir in "${MOD_DIRS[@]}"; do
    rel="${dir#$ROOT_DIR/}"
    echo "    $rel"
    (cd "$dir" && go mod tidy)
  done
fi

echo ""
echo "==> Done! Updated initia to $VERSION across ${#MOD_DIRS[@]} module(s)."
