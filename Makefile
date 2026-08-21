APP := gitwatch
GOFLAGS ?=

.PHONY: check test race vet fmt lint diff-check security performance build release \
	secrets secrets-history install-hooks clean

check: fmt lint test race vet diff-check security performance

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

diff-check:
	git diff --check
	git diff --cached --check

security:
	./scripts/security-check.sh

performance:
	./scripts/performance-check.sh

build:
	go build $(GOFLAGS) ./cmd/gitwatch

release:
	VERSION=$(VERSION) ./scripts/release.sh

secrets:
	./scripts/secret-scan.sh --staged

secrets-history:
	./scripts/secret-scan.sh --history

install-hooks:
	./scripts/install-hooks.sh

clean:
	rm -f $(APP)
