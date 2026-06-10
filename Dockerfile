FROM golang:1.23-alpine AS builder

WORKDIR /src

COPY go.mod .
COPY main.go .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o app .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /src/app .

EXPOSE 8080

ENTRYPOINT ["./app"]