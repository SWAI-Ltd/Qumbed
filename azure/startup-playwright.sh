#!/bin/bash
# Azure Web Apps startup script: install Playwright and browsers, then start the app.
# Set this as the startup command in App Service Configuration, or run from a custom container.

set -e

# App root (Azure Linux: /home/site/wwwroot; override with STARTUP_APP_PATH)
APP_PATH="${STARTUP_APP_PATH:-/home/site/wwwroot}"
cd "$APP_PATH"

echo "==> Installing dependencies..."
if [ -f "package.json" ]; then
  npm ci --omit=dev 2>/dev/null || npm install --omit=dev
fi

echo "==> Installing Playwright browsers..."
# Install Chromium only by default (smaller; add 'firefox' 'webkit' if needed)
npx playwright install chromium
# Optional: install system libs (needs root on some images; skip if not available)
npx playwright install-deps 2>/dev/null || true

echo "==> Starting application..."
# Override with STARTUP_CMD, e.g. "npm start" or "node dist/index.js"
if [ -n "$STARTUP_CMD" ]; then
  exec bash -c "$STARTUP_CMD"
fi
# Default: node server.js or npm start if no server.js
if [ -f "server.js" ]; then
  exec node server.js
fi
exec npm start
