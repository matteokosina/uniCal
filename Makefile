BINARY_NAME=unical

.PHONY: all build clean run config help

all: build

build:
	@echo "Building uniCal..."
	go build -o bin/$(BINARY_NAME) ./cmd/

clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -rf ical/

run:
	@echo "Running uniCal..."
	go run ./cmd/main.go

config:
	@echo "Starting configuration UI..."
	go run ./cmd/main.go config

install-deps:
	@echo "Installing dependencies..."
	go mod tidy

help:
	@echo "Available commands:"
	@echo "  make build     - Build both binaries"
	@echo "  make run       - Run the calendar filter"  
	@echo "  make config    - Run the configuration UI"
	@echo "  make clean     - Clean build artifacts"
	@echo "  make install-deps - Install Go dependencies"
