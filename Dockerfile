FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.mod
COPY go.sum go.sum

COPY ./internal ./internal
COPY ./app.go ./app.go

RUN CGO_ENABLED=0 GOOS=linux go build -mod vendor -o /server .

FROM alpine

COPY --from=builder /server ./server

CMD ["./server"]
