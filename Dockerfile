FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/google-health-bot .

# runtime
FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /app/

COPY --from=builder /app/google-health-bot .

ENTRYPOINT ["./google-health-bot"]
