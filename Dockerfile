# Stage 1: Build Go static binary (Pure Go, CGO_ENABLED=0)
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /app/server ./cmd/server

# Stage 2: Runtime leve com Chromium Headless e fontes pt-BR
FROM alpine:3.20
RUN apk add --no-cache \
    chromium \
    nss \
    freetype \
    harfbuzz \
    ca-certificates \
    ttf-freefont \
    font-noto \
    tzdata \
    wget

# O nome do binario varia entre versoes do pacote chromium no Alpine.
# Criamos um caminho estavel para CHROME_BIN/CHROME_PATH.
RUN if [ -x /usr/bin/chromium-browser ]; then \
      ln -sf /usr/bin/chromium-browser /usr/local/bin/chrome; \
    else \
      ln -sf /usr/bin/chromium /usr/local/bin/chrome; \
    fi && test -x /usr/local/bin/chrome

RUN addgroup -S app && adduser -S -G app app
WORKDIR /app
COPY --from=builder /app/server /app/server
RUN mkdir -p /app/data/pdfs && chown -R app:app /app

ENV PORT=8080 \
    DB_PATH=/app/data/contracts.db \
    LOCAL_PDF_DIR=/app/data/pdfs \
    CHROME_BIN=/usr/local/bin/chrome \
    CHROME_PATH=/usr/local/bin/chrome \
    CHROMEDP_NO_SANDBOX=true \
    TZ=America/Campo_Grande

USER app
EXPOSE 8080
VOLUME ["/app/data"]

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -qO- http://localhost:8080/health >/dev/null || exit 1

CMD ["/app/server"]
