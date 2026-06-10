.PHONY: build test start clean smoke docker-build docker-start docker-stop

build:
	go build -o tinygate .

test:
	go test ./... -v

start:
	@./tinygate -config config.yaml &

clean:
	rm -f tinygate

docker-build:
	CGO_ENABLED=0 GOOS=linux go build -o tinygate .
	docker build -t tinygate .

docker-start:
	@docker run -d -p 39901:39901 --env-file .env --name tinygate tinygate

docker-stop:
	@docker stop tinygate 2>/dev/null; docker rm tinygate 2>/dev/null; true

smoke:
	@./scripts/smoke-test.sh

# one command to rule them all
all: test build
	@echo "TinyGate ready. Run 'make start' to launch."
