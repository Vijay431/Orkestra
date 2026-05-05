FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o orkestra ./cmd/server

# Debug target — keeps shell for troubleshooting
FROM alpine:3.19 AS debug
RUN apk add --no-cache ca-certificates sqlite
COPY --from=builder /app/orkestra /orkestra
COPY skill/SKILL.md /ORKESTRA_SKILL.md
VOLUME ["/data"]
EXPOSE 8080
EXPOSE 7777
ENTRYPOINT ["/orkestra"]

# Production target — minimal scratch image
FROM scratch
COPY --from=builder /app/orkestra /orkestra
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY skill/SKILL.md /ORKESTRA_SKILL.md
VOLUME ["/data"]
EXPOSE 8080
EXPOSE 7777
ENTRYPOINT ["/orkestra"]
