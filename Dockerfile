# syntax=docker/dockerfile:1

# This service uses github.com/gen2brain/go-fitz (MuPDF), which requires CGO + system libs.
# Cloud Run deployment target: linux/amd64 (default) - adjust GOARCH if needed.

FROM golang:1.24-bookworm AS builder

WORKDIR /src

# Build deps for go-fitz / MuPDF
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    ca-certificates \
    pkg-config \
    libmupdf-dev \
  && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -trimpath -o /out/server ./cmd/server

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    libmupdf-dev \
  && rm -rf /var/lib/apt/lists/*

# Run as non-root (recommended for Cloud Run)
RUN useradd -u 10001 -m appuser

WORKDIR /app
COPY --from=builder /out/server /app/server

ENV PORT=8080
EXPOSE 8080

USER 10001
CMD ["/app/server"]

