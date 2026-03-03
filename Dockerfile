FROM golang:1.25-alpine AS builder

WORKDIR /bh_lib-app

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o bh_lib-bin ./cmd/main.go

FROM alpine:3.22
RUN apk add --no-cache postgresql-client curl

RUN mkdir -p /app/instruction

WORKDIR /app

COPY --from=builder /bh_lib-app/bh_lib-bin /app/bh_lib-bin

COPY instruction /app/instruction

ENTRYPOINT ["/app/bh_lib-bin"]
