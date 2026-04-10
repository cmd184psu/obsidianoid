# Obsidianoid Makefile
# Builds, installs, and manages the obsidianoid systemd service.

BINARY     := obsidianoid
BIN_DIR    := /usr/local/bin
APP_DIR    := /opt/obsidianoid
STATIC_DIR := $(APP_DIR)/static
SVC_FILE   := obsidianoid.service
SVC_DIR    := /etc/systemd/system
GO         := go

.PHONY: all build install uninstall start stop restart status logs clean test

## Default: build the binary
all: build

## Compile the binary for the current platform
build:
	$(GO) build -o $(BINARY) ./cmd/obsidianoid/
	@echo "Built ./$(BINARY)"

## Cross-compile for Raspberry Pi (arm64)
build-pi:
	GOOS=linux GOARCH=arm64 $(GO) build -o $(BINARY)-arm64 ./cmd/obsidianoid/
	@echo "Built ./$(BINARY)-arm64 (Raspberry Pi arm64)"

## Cross-compile for Raspberry Pi older models (arm v6/v7)
build-pi-armv7:
	GOOS=linux GOARCH=arm GOARM=7 $(GO) build -o $(BINARY)-armv7 ./cmd/obsidianoid/
	@echo "Built ./$(BINARY)-armv7 (Raspberry Pi armv7)"

## Run tests with coloured pass/fail output
test:
	$(GO) test ./... -v 2>&1 | awk '\
		/--- PASS/ { print "\033[32m" $$0 "\033[0m"; next } \
		/--- FAIL/ { print "\033[31m" $$0 "\033[0m"; next } \
		{ print }'

## Install binary + static files + systemd service, then enable and start
install: build
	@echo ">>> Installing binary to $(BIN_DIR)/$(BINARY)"
	install -m 0755 $(BINARY) $(BIN_DIR)/$(BINARY)

	@echo ">>> Ensuring $(STATIC_DIR) exists"
	mkdir -p $(STATIC_DIR)

	@echo ">>> Copying static assets"
	cp -r static/. $(STATIC_DIR)/

	@echo ">>> Installing systemd service"
	install -m 0644 $(SVC_FILE) $(SVC_DIR)/$(SVC_FILE)

	@echo ">>> Reloading systemd daemon"
	systemctl daemon-reload

	@echo ">>> Enabling and starting obsidianoid"
	systemctl enable $(BINARY)
	systemctl start  $(BINARY)

	@echo ""
	@echo "\033[32m✓ obsidianoid installed and running.\033[0m"
	@echo "  Check status : sudo systemctl status obsidianoid"
	@echo "  View logs    : sudo journalctl -u obsidianoid -f"

## Stop and remove the service, binary, and static files
uninstall:
	@echo ">>> Stopping and disabling service"
	-systemctl stop    $(BINARY)
	-systemctl disable $(BINARY)
	@echo ">>> Removing service file"
	-rm -f $(SVC_DIR)/$(SVC_FILE)
	@echo ">>> Reloading systemd daemon"
	-systemctl daemon-reload
	@echo ">>> Removing binary"
	-rm -f $(BIN_DIR)/$(BINARY)
	@echo "\033[33m✓ Uninstalled. Static files and config are untouched.\033[0m"
	@echo "  Remove manually: rm -rf $(APP_DIR)  ~/.obsidianoid  ~/.obsidianoid.json"

start:
	systemctl start $(BINARY)

stop:
	systemctl stop $(BINARY)

restart:
	systemctl restart $(BINARY)

status:
	systemctl status $(BINARY)

logs:
	journalctl -u $(BINARY) -f

clean:
	rm -f $(BINARY) $(BINARY)-arm64 $(BINARY)-armv7
