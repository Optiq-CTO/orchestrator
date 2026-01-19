FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY . .
RUN go build -mod=vendor -o server ./cmd/server

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/server .

CMD ["./server"]
