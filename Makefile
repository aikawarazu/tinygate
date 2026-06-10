.PHONY: build test start clean

build:
	go build -o tinygate .

test:
	go test ./... -v

start:
	@./tinygate -config config.yaml &

clean:
	rm -f tinygate

# one command to rule them all
all: test build
	@echo "TinyGate ready. Run 'make start' to launch."
