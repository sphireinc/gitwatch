APP := gitwatch
GOFLAGS ?=
GOLANGCI_LINT_VERSION := $(shell cat .golangci-lint-version)
# Repository tests create disposable commits. Keep host-wide signing settings
# from changing deterministic test behavior; production Git runners retain the
# user's normal configuration.
GIT_TEST_ENV := GIT_CONFIG_NOSYSTEM=1 GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_COUNT=1 GIT_CONFIG_KEY_0=commit.gpgsign GIT_CONFIG_VALUE_0=false

.PHONY: check test race vet fmt lint diff-check security performance build release \
	secrets secrets-history install-hooks clean

check: fmt lint test race vet diff-check security performance

test:
	$(GIT_TEST_ENV) go test ./...

race:
	$(GIT_TEST_ENV) go test -race ./...

vet:
	go vet ./...

fmt:
	test -z "$$(gofmt -l .)"

lint:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run

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
