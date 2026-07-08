# Makefile for agent-compose
BINARY := ac
MODULE := github.com/zanetworker/agent-compose

.PHONY: build test test-race clean install

build:
	go build -o $(BINARY) ./cmd/ac/

test:
	go test ./pkg/compose/ -v -count=1

test-race:
	go test ./pkg/compose/ -v -race -count=1

clean:
	rm -f $(BINARY)

install:
	go install ./cmd/ac/
