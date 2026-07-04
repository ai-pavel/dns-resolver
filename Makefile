.PHONY: build clean test

BINARY=bin/resolve

build:
	go build -o $(BINARY) ./cmd/resolve

clean:
	rm -rf bin/

test:
	go test ./...

run: build
	./$(BINARY) $(ARGS)
