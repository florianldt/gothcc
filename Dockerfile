# -------------------------------------------------------
# Stage 1: Build Tailwind CSS (no Node)
# -------------------------------------------------------
FROM alpine:3.21 AS css
ARG TARGETARCH
RUN apk add --no-cache curl
WORKDIR /app
RUN ARCH=$(case "$TARGETARCH" in arm64) echo arm64 ;; *) echo x64 ;; esac) \
    && curl -sL "https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-${ARCH}" \
    -o /usr/local/bin/tailwindcss && chmod +x /usr/local/bin/tailwindcss
COPY styles/ styles/
COPY internal/components/ internal/components/
RUN mkdir -p static/css && tailwindcss -i styles/app.css -o static/css/app.css --minify

# -------------------------------------------------------
# Stage 2: Build the Go binary (uses committed generated code — no codegen here)
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
COPY --from=css /app/static/css/app.css ./static/css/app.css
# migrations are embedded in the binary via go:embed — nothing else to copy.
ENV LOG_LEVEL=info
EXPOSE 8000
USER nonroot:nonroot
ENTRYPOINT ["./server"]
