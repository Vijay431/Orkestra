#!/usr/bin/env bash
# Regenerate vendored static assets. Run manually when Alpine or Tailwind
# classes used in index.html change. Does NOT run at server startup.
#
# Requirements: Node 18+ with npm on PATH.
#
# Pinned versions:
#   Alpine.js  3.14.9
#   Tailwind   3.x (latest 3.x via npm)

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
INDEX_HTML="$SCRIPT_DIR/../index.html"

echo "Downloading Alpine.js 3.14.9..."
curl -fsSL -o "$SCRIPT_DIR/alpine.min.js" \
  "https://unpkg.com/alpinejs@3.14.9/dist/cdn.min.js"

echo "Generating Tailwind CSS..."
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

cat > "$TMPDIR/input.css" << 'EOF'
@tailwind base;
@tailwind components;
@tailwind utilities;

@keyframes fadeInUp {
  from { opacity: 0; transform: translateY(8px); }
  to   { opacity: 1; transform: translateY(0); }
}
.card-enter {
  opacity: 0;
  animation: fadeInUp 300ms ease-out forwards;
}
@keyframes spin360 {
  to { transform: rotate(360deg); }
}
.spin-anim { animation: spin360 0.8s linear infinite; }
.line-clamp-2 {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.col-scroll::-webkit-scrollbar { width: 6px; }
.col-scroll::-webkit-scrollbar-thumb { background: #d1d5db; border-radius: 3px; }
.col-scroll::-webkit-scrollbar-track { background: transparent; }
EOF

npm install --prefix "$TMPDIR" tailwindcss@3 --no-save --silent

"$TMPDIR/node_modules/.bin/tailwindcss" \
  -i "$TMPDIR/input.css" \
  -o "$SCRIPT_DIR/tailwind.css" \
  --content "$INDEX_HTML" \
  --minify

echo "Done. Commit internal/web/static/vendor/ with any changes."
