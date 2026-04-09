.PHONY: build run clean test install

BINARY_NAME=electric-grid-monitor
BUILD_DIR=./build
INSTALL_DIR=/usr/local/bin
SERVICE_NAME=electric-grid-monitor

build:
	mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) cmd/monitor/main.go

run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

dev:
	go run cmd/monitor/main.go

clean:
	rm -rf $(BUILD_DIR)
	rm -f data/*.db

test:
	go test -v ./...

install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_DIR)..."
	cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/
	@echo "Installing systemd service..."
	cp systemd/$(SERVICE_NAME).service /etc/systemd/system/
	@echo "Creating directories..."
	mkdir -p /etc/$(SERVICE_NAME)
	mkdir -p /var/lib/$(SERVICE_NAME)/data
	mkdir -p /var/log/$(SERVICE_NAME)
	@echo "Installation complete!"
	@echo "Next steps:"
	@echo "  1. Copy .env to /etc/$(SERVICE_NAME)/.env and configure"
	@echo "  2. Run: systemctl daemon-reload"
	@echo "  3. Run: systemctl enable $(SERVICE_NAME)"
	@echo "  4. Run: systemctl start $(SERVICE_NAME)"

uninstall:
	systemctl stop $(SERVICE_NAME) || true
	systemctl disable $(SERVICE_NAME) || true
	rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	rm -f /etc/systemd/system/$(SERVICE_NAME).service
	systemctl daemon-reload

fmt:
	go fmt ./...

lint:
	golangci-lint run

deps:
	go mod tidy
	go mod download
