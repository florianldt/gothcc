.PHONY: setup templ sqlc generate css css-watch dev build run check

# Resolve the standalone Tailwind CLI asset for this OS/arch.
UNAME_S := $(shell uname -s | tr '[:upper:]' '[:lower:]')
UNAME_M := $(shell uname -m)
ifeq ($(UNAME_S),darwin)
	TW_OS := macos
else
	TW_OS := linux
endif
ifeq ($(UNAME_M),arm64)
	TW_ARCH := arm64
else ifeq ($(UNAME_M),aarch64)
	TW_ARCH := arm64
else
	TW_ARCH := x64
endif
TW_ASSET := tailwindcss-$(TW_OS)-$(TW_ARCH)

# One-time: download the standalone Tailwind CLI (no Node) + install sqlc.
# templ is pinned as a Go tool directive, so it needs no global install.
setup:
	curl -sLo tailwindcss "https://github.com/tailwindlabs/tailwindcss/releases/latest/download/$(TW_ASSET)"
	chmod +x tailwindcss
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

templ:
	go tool templ generate

sqlc:
	sqlc generate

generate: templ sqlc

css:
	./tailwindcss -i styles/app.css -o static/css/app.css --minify

css-watch:
	./tailwindcss -i styles/app.css -o static/css/app.css --watch

# Full dev loop: Tailwind watches CSS while templ regenerates, rebuilds, and
# runs a live-reload proxy on :7331 in front of the app on :8000.
dev:
	./tailwindcss -i styles/app.css -o static/css/app.css --watch & \
	go tool templ generate --watch --proxy="http://localhost:8000" --cmd="go run ."

build: generate css
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/server .

run:
	go run .

# Fail if generated code is stale or the build is unhealthy.
check: generate
	git diff --exit-code
	go vet ./...
