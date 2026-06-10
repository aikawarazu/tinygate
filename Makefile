.PHONY: build test start clean docker-build docker-start

build:
	go build -o tinygate .

test:
	go test ./... -v

start:
	@./tinygate -config config.yaml &

clean:
	rm -f tinygate

docker-build:
	docker build -t tinygate .

docker-start:
	@docker run -d -p 39901:39901 --env-file .env --name tinygate tinygate

docker-stop:
	@docker stop tinygate 2>/dev/null; docker rm tinygate 2>/dev/null; true

# one command to rule them all
all: test build
	@echo "TinyGate ready. Run 'make start' to launch."
