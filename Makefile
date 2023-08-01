.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: test-deps
test-deps:
	cd testdata/src/a && go mod vendor

.PHONY: test
test: test-deps
	go test -v -covermode=atomic -coverprofile=cover.out -coverpkg ./... ./...

.PHONY: build
build:
	go build -o bin/tracecheck ./cmd/tracecheck

.PHONY: build-plugin
build-plugin:
	CGO_ENABLED=1 go build -o bin/tracecheck.so -buildmode=plugin ./plugin

.PHONY: build-all
build-all: build build-plugin
