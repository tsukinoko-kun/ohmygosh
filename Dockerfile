FROM golang:1-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 go build -ldflags="-s -w -X github.com/tsukinoko-kun/ohmygosh/internal/metadata.Version=$Version" -o ohmygosh .

FROM alpine:latest
WORKDIR /usr/local/bin
COPY --from=builder /app/ohmygosh .
WORKDIR /
ENTRYPOINT [ "/usr/local/bin/ohmygosh" ]
