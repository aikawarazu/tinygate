FROM nginx:1.16.0-alpine

WORKDIR /app

COPY tinygate .
COPY config.yaml .

EXPOSE 39901

CMD ["./tinygate", "-config", "config.yaml"]
