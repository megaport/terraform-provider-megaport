default: build

build:
	go build -v ./...

install:
	go install -v ./...

test:
	go test -v -timeout=30m -cover ./internal/provider/

testacc:
ifndef MEGAPORT_ACCESS_KEY
	$(error MEGAPORT_ACCESS_KEY is not set)
endif
ifndef MEGAPORT_SECRET_KEY
	$(error MEGAPORT_SECRET_KEY is not set)
endif
	TF_ACC=1 go test -v -timeout=30m -cover ./internal/provider/

lint:
	golangci-lint run --timeout=10m

generate:
	go generate ./...

fmt:
	gofmt -w .
	@command -v terraform >/dev/null 2>&1 && terraform fmt -recursive examples/ || echo "Skipping terraform fmt: terraform CLI not found"

clean:
	go clean
	rm -rf docs/

.PHONY: default build install test testacc lint generate fmt clean
