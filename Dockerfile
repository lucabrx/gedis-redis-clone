FROM golang:1.24.4-alpine AS builder

WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o gedis .

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/gedis .
EXPOSE 6379

CMD ["./gedis", "-port", "6379", "-aof", "/data/database.aof"]
