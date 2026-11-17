GO ?= go

.PHONY: check fmt vet test build

check: fmt vet test build

fmt:
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

vet:
	CGO_ENABLED=0 $(GO) vet ./...

test:
	CGO_ENABLED=0 $(GO) test -count=1 ./...

build:
	@mkdir -p bin
	CGO_ENABLED=0 $(GO) build -trimpath -o bin/aifactory ./cmd/aifactory
