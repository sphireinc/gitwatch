APP := gitwatch
GOFLAGS ?=

.PHONY: test race vet fmt lint build release clean

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

fmt:
	test -z "$$(gofmt -l .)"

lint:
	golangci-lint run

build:
	go build $(GOFLAGS) ./cmd/gitwatch

release:
	VERSION=$(VERSION) ./scripts/release.sh

clean:
	rm -f $(APP)
