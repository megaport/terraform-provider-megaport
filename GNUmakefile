default: build

build:
	go build -v ./...

install:
	go install -v ./...

test:
	go test -v -timeout=30m -cover ./internal/provider/

testacc:
	TF_ACC=1 go test -v -timeout=30m -cover ./internal/provider/

lint:
	golangci-lint run --timeout=10m

generate:
	go generate ./...

fmt:
	gofmt -w .
	terraform fmt -recursive examples/

clean:
	go clean

.PHONY: build install test testacc lint generate fmt clean
