FROM golang:1.26-alpine AS builder

RUN apk --no-cache add ca-certificates git tzdata gcc musl-dev libwebp-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s -extldflags '-static'" -o bin/server ./cmd/server

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /app/bin/server /server

EXPOSE 8080

ENTRYPOINT ["/server"]
