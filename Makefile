.PHONY: build test vet install uninstall clean

BINARY := note
BIN_DIR := $(HOME)/.local/bin

build:
	go build -buildvcs=false -o bin/$(BINARY) .

test:
	go test ./...

vet:
	go vet ./...

install: build
	install -d $(BIN_DIR)
	install -m 0755 bin/$(BINARY) $(BIN_DIR)/$(BINARY)

uninstall:
	rm -f $(BIN_DIR)/$(BINARY)

clean:
	rm -rf bin
