FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY tinygate .
COPY config.yaml .

EXPOSE 39901

CMD ["./tinygate", "-config", "config.yaml"]
