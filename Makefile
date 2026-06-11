GO := /usr/local/go/bin/go

.PHONY: build test start clean smoke models docker-build docker-start docker-stop package

build:
	$(GO) build -o tinygate .

test:
	$(GO) test ./... -v

start:
	@./tinygate -config config.yaml &

clean:
	rm -f tinygate

package: build
	@mkdir -p dist
	tar czf dist/tinygate.tar.gz tinygate config.yaml .env.example
	@echo "dist/tinygate.tar.gz ready"

models:
	@./scripts/models-ref.sh

smoke:
	@./scripts/smoke-test.sh

docker-build:
	CGO_ENABLED=0 GOOS=linux $(GO) build -o tinygate .
	docker build -t tinygate .

docker-start:
	@docker run -d -p 39901:39901 --env-file .env --name tinygate tinygate

docker-stop:
	@docker stop tinygate 2>/dev/null; docker rm tinygate 2>/dev/null; true

# one command to rule them all
all: test build
	@echo "TinyGate ready. Run 'make start' to launch."
