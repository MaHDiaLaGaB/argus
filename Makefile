# Same verbs as every Argus template: setup, build, test, test-one, lint, fmt, run.
BIN := bin/argus
PKG := ./...

.PHONY: setup build test test-one lint fmt run clean

setup:
	go mod download

build:
	go build -trimpath -o $(BIN) ./cmd/argus

test:
	go test -race -count=1 $(PKG)

# make test-one T=TestWorkTicketValidate
test-one:
	go test -race -count=1 -run '$(T)' $(PKG)

lint:
	@unformatted=$$(gofmt -l .); if [ -n "$$unformatted" ]; then echo "gofmt needed:"; echo "$$unformatted"; exit 1; fi
	go vet $(PKG)
	@if command -v golangci-lint >/dev/null 2>&1; then golangci-lint run; else echo "golangci-lint not installed, skipped (CI runs it)"; fi

fmt:
	gofmt -w .

run: build
	./$(BIN) $(ARGS)

clean:
	rm -rf bin
