BINARY := orcai
PROTO_DIR := proto/orcai/v1
PROTO_OUT := proto/orcai/v1

.PHONY: proto build run test clean

all: build run

run: build
	-tmux kill-session -t orcai 2>/dev/null
	bin/$(BINARY)

proto:
	PATH="$$PATH:$$(go env GOPATH)/bin" protoc \
		--go_out=$(PROTO_OUT) --go_opt=paths=source_relative \
		--go-grpc_out=$(PROTO_OUT) --go-grpc_opt=paths=source_relative \
		-I proto/orcai/v1 \
		proto/orcai/v1/plugin.proto \
		proto/orcai/v1/bus.proto

build:
	go build -o bin/$(BINARY) .

test:
	go test ./...

clean:
	rm -f bin/$(BINARY)
