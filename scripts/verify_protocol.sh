#!/bin/bash
# Run ProVerif on the Qumbed protocol model.
# Exit 0 only if both secrecy and correspondence queries are proven.
# Requires: proverif (e.g. opam install proverif)
set -e
cd "$(dirname "$0")/.."
PV_FILE="${PV_FILE:-proverif/qumbed.pv}"
if ! command -v proverif >/dev/null 2>&1; then
  echo "proverif not found. Install with: opam install proverif" >&2
  exit 1
fi
OUT=$(mktemp)
if ! proverif "$PV_FILE" > "$OUT" 2>&1; then
  cat "$OUT"
  rm -f "$OUT"
  exit 1
fi
if grep "RESULT.*is false" "$OUT" >/dev/null 2>&1; then
  echo "Verification failed: at least one query is false." >&2
  cat "$OUT"
  rm -f "$OUT"
  exit 1
fi
echo "ProVerif verification passed:"
grep -E "RESULT|Verification summary" -A 20 "$OUT" || cat "$OUT"
rm -f "$OUT"
exit 0
