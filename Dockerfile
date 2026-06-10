FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o tinygate .

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/tinygate .
COPY --from=builder /app/config.yaml .

EXPOSE 39901

CMD ["./tinygate", "-config", "config.yaml"]
