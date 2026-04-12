.PHONY: lint test

lint:
	golangci-lint run ./...

test:
	go test -race -count=1 -timeout 120s ./...

vet:
	go vet ./...

all: vet lint test
