FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod .
COPY go.sum .
COPY app.go .

RUN CGO_ENABLED=0 GOOS=linux go build -o /server .

FROM alpine

COPY --from=builder /server ./server

CMD ["./server"]
