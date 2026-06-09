FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY main.go .

RUN go build -o app main.go

FROM alpine:3.20

WORKDIR /app
COPY --from=builder /app/app .

EXPOSE 8080

CMD ["./app"]