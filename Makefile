.PHONY: build run clean install test

# Binary name
BINARY_NAME=gofileyourself

# Build directory
BUILD_DIR=bin

# Main package path
MAIN_PATH=./cmd/gofileyourself

# Build the binary
build:
	mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

# Run the application
run:
	go run $(MAIN_PATH)

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)

# Install globally
install:
	go install $(MAIN_PATH)
