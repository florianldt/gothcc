#!/bin/sh
set -e

cd /app

mkdir -p static/css

echo "==> initial codegen"
go tool templ generate
tailwindcss -i styles/app.css -o static/css/app.css

# Tailwind watches in the background; it only writes static/css/app.css, which
# nothing else rebuilds on. Keep watching even when Docker closes stdin.
echo "==> watching css"
tailwindcss -i styles/app.css -o static/css/app.css --watch=always &

# air runs in the foreground so its output is the container log and a crash
# takes the container with it. It regenerates templ as part of each build
# (see .air.toml), so .templ and .go edits both trigger one rebuild.
echo "==> starting air"
exec air
