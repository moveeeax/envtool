BINARY := envtool
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X github.com/moveeeax/envtool/cmd.version=$(VERSION)

.PHONY: build test vet fmt clean

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -s -w .

clean:
	rm -f $(BINARY)
