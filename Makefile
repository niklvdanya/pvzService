APP_NAME := pvz
BUILD_DIR := bin
MAIN_PATH := cmd/cli/main.go

.PHONY: update linter build start run clean

update:
	go mod tidy
	go mod download

linter:
	go vet ./...
	go fmt ./...
	golangci-lint run; 

build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)

start:
	@if [ ! -f $(BUILD_DIR)/$(APP_NAME) ]; then \
		echo "Бинарный файл не найден. Сначала выполните 'make build'"; \
		exit 1; \
	fi
	./$(BUILD_DIR)/$(APP_NAME)

run: update linter build start


generate-data:
	go run data/generate_test_data.go
