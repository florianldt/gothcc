# -------------------------------------------------------
# Stage 1: Build Tailwind CSS (no Node)
# -------------------------------------------------------
FROM alpine:3.21 AS css
ARG TARGETARCH
# Keep this Tailwind version in sync with dev/Dockerfile.
ARG TAILWIND_VERSION=v4.3.3
# curl to fetch assets; libstdc++/libgcc are runtime deps of the musl Tailwind build.
RUN apk add --no-cache curl libstdc++ libgcc
WORKDIR /app
# Alpine is musl-based, so use Tailwind's -musl standalone build (the plain
# linux build is glibc-linked and fails to exec here as "not found").
RUN ARCH=$(case "$TARGETARCH" in arm64) echo arm64 ;; *) echo x64 ;; esac) \
    && curl -fsSL "https://github.com/tailwindlabs/tailwindcss/releases/download/${TAILWIND_VERSION}/tailwindcss-linux-${ARCH}-musl" \
    -o /usr/local/bin/tailwindcss && chmod +x /usr/local/bin/tailwindcss \
    && tailwindcss --help >/dev/null
COPY styles/ styles/
COPY internal/components/ internal/components/
RUN mkdir -p static/css && tailwindcss -i styles/app.css -o static/css/app.css --minify

# -------------------------------------------------------
# Stage 2: Build the Go binary (uses committed generated code, no codegen here)
# -------------------------------------------------------
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/server .

# -------------------------------------------------------
# Stage 3: Minimal, non-root runtime
# -------------------------------------------------------
FROM gcr.io/distroless/static:nonroot AS runtime
WORKDIR /app
COPY --from=build /out/server ./server
# Everything under static/ is served from disk at /static; the stylesheet
# comes from the css stage. Migrations are embedded in the binary.
COPY --from=build /src/static ./static
COPY --from=css /app/static/css/app.css ./static/css/app.css
ENV LOG_LEVEL=info
EXPOSE 8000
USER nonroot:nonroot
ENTRYPOINT ["./server"]
