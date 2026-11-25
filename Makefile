.PHONY: build run test clean

build:
	go build -o zuelose ./cmd/zuelose

run:
	go run ./cmd/zuelose

test:
	go test ./...

clean:
	rm -f zuelose
