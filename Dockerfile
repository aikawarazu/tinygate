FROM golang:1.21-alpine

WORKDIR /app

COPY tinygate .
COPY config.yaml .

EXPOSE 39901

CMD ["./tinygate", "-config", "config.yaml"]
