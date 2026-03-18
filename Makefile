VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BINARY      := hm
INSTALL_DIR := $(HOME)/.local/bin

.PHONY: build test install clean

build:
	go build -ldflags="-X main.version=$(VERSION)" -o $(BINARY) .

test:
	go test ./...

install: build
	mkdir -p $(INSTALL_DIR)
	cp $(BINARY) $(INSTALL_DIR)/$(BINARY)
	@echo "Installed to $(INSTALL_DIR)/$(BINARY)"
	@echo "Ensure $(INSTALL_DIR) is in your PATH"

clean:
	rm -f $(BINARY)
