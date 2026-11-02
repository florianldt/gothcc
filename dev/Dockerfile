# Development image. The production image is the multi-stage distroless build
# in ../Dockerfile. This one is the opposite: it carries the whole toolchain
# and expects the source to be bind-mounted, so edits on the host rebuild
# inside the container.
FROM golang:1.26-alpine

ARG TARGETARCH
# Pinned, not @latest. A dev container that drifts is a dev container that
# reproduces bugs the host doesn't have, and sqlc in particular must match the
# version that generated the committed code, or `make check` fails in CI.
ARG AIR_VERSION=v1.67.4
ARG SQLC_VERSION=v1.31.1
# Keep this Tailwind version in sync with the production Dockerfile.
ARG TAILWIND_VERSION=v4.3.3

# libstdc++/libgcc are for the Tailwind binary below, which needs them even in
# its musl build and which Alpine does not ship by default.
RUN apk add --no-cache curl git make libstdc++ libgcc

RUN go install github.com/air-verse/air@${AIR_VERSION} && \
    go install github.com/sqlc-dev/sqlc/cmd/sqlc@${SQLC_VERSION}

# Arch-aware, so this works on Apple Silicon as well as x86. The -musl variant
# is required: this is an Alpine image, and the default build is linked against
# glibc: it downloads fine and then fails to exec with a bare "not found",
# because what is missing is the dynamic loader, not the file.
# The smoke test keeps that failure at build time instead of at startup.
RUN ARCH=$(case "$TARGETARCH" in arm64) echo arm64 ;; *) echo x64 ;; esac) \
    && curl -fsSL "https://github.com/tailwindlabs/tailwindcss/releases/download/${TAILWIND_VERSION}/tailwindcss-linux-${ARCH}-musl" \
       -o /usr/local/bin/tailwindcss \
    && chmod +x /usr/local/bin/tailwindcss \
    && tailwindcss --help >/dev/null

# The bind-mounted repo is owned by the host user; let `make check` run git
# diff on it from inside the container.
RUN git config --global --add safe.directory /app

WORKDIR /app

# Warm the module cache at build time so a cold `make dev` doesn't sit
# downloading. templ needs no install: go.mod pins it as a tool directive.
COPY go.mod go.sum ./
RUN go mod download

COPY dev/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

EXPOSE 8000
ENTRYPOINT ["entrypoint.sh"]
