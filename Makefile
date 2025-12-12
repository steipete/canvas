.PHONY: fmt lint test

fmt:
	gofmt -w .

lint:
	golangci-lint run

test:
	go test ./...

