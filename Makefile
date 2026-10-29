
.PHONY: run fmt fmt-check test vet check migrate-up migrate-down

GO_MODULE_DIR := src

run:
	cd $(GO_MODULE_DIR) && go run ./cmd/server

fmt:
	cd $(GO_MODULE_DIR) && gofmt -w $$(find . -type f -name '*.go' -not -path './vendor/*')

fmt-check:
	cd $(GO_MODULE_DIR) && \
		test -z "$$(gofmt -l .)" || \
		(echo "Go files need formatting. Run 'make fmt'." && exit 1)

test:
	cd $(GO_MODULE_DIR) && go test ./...

vet:
	cd $(GO_MODULE_DIR) && go vet ./...

check: fmt-check test vet

migrate-up:
	cd src && go run ./cmd/migrate up

migrate-down:
	cd src && go run ./cmd/migrate down