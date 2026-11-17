# Stage 1: Build Go binary (Pure Go, CGO_ENABLED=0)
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /ia-infinito-contracts ./cmd/server

# Stage 2: Runtime with Chromium headless
FROM alpine:3.20
RUN apk add --no-cache chromium nss freetype harfbuzz ca-certificates ttf-freefont
ENV CHROME_PATH=/usr/bin/chromium-browser
WORKDIR /app
COPY --from=builder /ia-infinito-contracts /app/ia-infinito-contracts
RUN mkdir -p /app/data/pdfs
EXPOSE 8080
VOLUME ["/app/data"]
CMD ["/app/ia-infinito-contracts"]
