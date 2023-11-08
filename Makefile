HOSTNAME=registry.terraform.io
NAMESPACE=megaport
NAME=megaport
BINARY=terraform-provider-${NAME}
VERSION=0.4.0
OS_ARCH=$$(go version | cut -d" " -f4 | sed 's/\//_/g')
ZIP_FILE=terraform-provider-${NAME}_${VERSION}_${OS_ARCH}.zip

default: install

clean:
	rm -rf ~/.terraform.d/plugins/${HOSTNAME}

build:
	mkdir -p ./bin/
	go build -o ./bin/${BINARY}

release:
	mkdir -p ./bin/
	GOOS=darwin GOARCH=amd64 go build -o ./bin/${BINARY}_${VERSION}_darwin_amd64
	GOOS=darwin GOARCH=arm64 go build -o ./bin/${BINARY}_${VERSION}_darwin_arm64
	GOOS=freebsd GOARCH=386 go build -o ./bin/${BINARY}_${VERSION}_freebsd_386
	GOOS=freebsd GOARCH=amd64 go build -o ./bin/${BINARY}_${VERSION}_freebsd_amd64
	GOOS=freebsd GOARCH=arm go build -o ./bin/${BINARY}_${VERSION}_freebsd_arm
	GOOS=linux GOARCH=386 go build -o ./bin/${BINARY}_${VERSION}_linux_386
	GOOS=linux GOARCH=amd64 go build -o ./bin/${BINARY}_${VERSION}_linux_amd64
	GOOS=linux GOARCH=arm go build -o ./bin/${BINARY}_${VERSION}_linux_arm
	GOOS=openbsd GOARCH=386 go build -o ./bin/${BINARY}_${VERSION}_openbsd_386
	GOOS=openbsd GOARCH=amd64 go build -o ./bin/${BINARY}_${VERSION}_openbsd_amd64
	GOOS=solaris GOARCH=amd64 go build -o ./bin/${BINARY}_${VERSION}_solaris_amd64
	GOOS=windows GOARCH=386 go build -o ./bin/${BINARY}_${VERSION}_windows_386
	GOOS=windows GOARCH=amd64 go build -o ./bin/${BINARY}_${VERSION}_windows_amd64

install: build
	mkdir -p ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}-local/${VERSION}/${OS_ARCH}
	mv bin/${BINARY} ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}-local/${VERSION}/${OS_ARCH}/${BINARY}-local
