FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY tinygate .
COPY config.yaml .

ENV TINYGATE_API_KEYS=""
ENV ZHIPU_API_KEY=""
ENV MIMO_API_KEY=""
ENV OPENCODE_GO_API_KEY=""

EXPOSE 39901

CMD ["./tinygate", "-config", "config.yaml"]
