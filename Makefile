BINARY := orcai

.PHONY: build run test clean debug-build debug debug-connect debug-tmux

all: build run

run: build
	-tmux kill-session -t orcai 2>/dev/null
	bin/$(BINARY)

build:
	go build -o bin/$(BINARY) .
	go build -o bin/orcai-welcome ./cmd/orcai-welcome/
	go build -o bin/orcai-picker ./cmd/orcai-picker/
	go build -o bin/orcai-sysop ./cmd/orcai-sysop/

test:
	go test ./...

clean:
	rm -f bin/$(BINARY) bin/$(BINARY)-debug bin/orcai-welcome bin/orcai-picker bin/orcai-sysop

debug-build:
	go build -gcflags="all=-N -l" -o bin/$(BINARY)-debug .

debug: debug-build
	@echo "Delve listening on :2345 — connect with: make debug-connect"
	dlv exec --headless --listen=:2345 --api-version=2 ./bin/$(BINARY)-debug

debug-connect:
	dlv connect :2345

debug-tmux: debug-build
	@bash $(shell pwd)/scripts/debug-tmux.sh
