VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BINARY      := hm
INSTALL_DIR := $(HOME)/.local/bin

.PHONY: build test vet install clean

build:
	go build -ldflags="-X main.version=$(VERSION)" -o $(BINARY) .

vet:
	go vet ./...

test: vet
	go test ./...

install: build
	mkdir -p $(INSTALL_DIR)
	cp $(BINARY) $(INSTALL_DIR)/$(BINARY)
	ln -sf $(INSTALL_DIR)/$(BINARY) $(INSTALL_DIR)/hmq
	@echo "Installed to $(INSTALL_DIR)/$(BINARY) (symlink: $(INSTALL_DIR)/hmq)"
	@echo "Ensure $(INSTALL_DIR) is in your PATH"

clean:
	rm -f $(BINARY)
