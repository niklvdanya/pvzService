ifneq ("$(wildcard tutor_makefile.mk)","")
include tutor_makefile.mk
endif
# пожалуйста, не удаляйте и не перемещайте этот импорт, он помогает вашему верному тьютору быстрее смотреть ваше дз
# вы можете описать ваш собственный makefile ниже

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

LOCAL_BIN := $(CURDIR)/bin
OUT_PATH := $(CURDIR)/pkg

bin-deps: export GOBIN := $(LOCAL_BIN)
bin-deps: export PROTOC_VERSION := protoc-31.1-linux-x86_64
bin-deps:
	curl -LO https://github.com/protocolbuffers/protobuf/releases/download/v31.1/$(PROTOC_VERSION).zip
	unzip -o $(PROTOC_VERSION).zip -d $(LOCAL_BIN)
	rm $(PROTOC_VERSION).zip

	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/envoyproxy/protoc-gen-validate@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest

generate: export GOBIN := $(LOCAL_BIN)
generate:
	mkdir -p $(OUT_PATH)
	$(LOCAL_BIN)/bin/protoc --proto_path=api --proto_path=vendor.protogen \
		--go_out=$(OUT_PATH) --go_opt=paths=source_relative --plugin protoc-gen-go="${GOBIN}/protoc-gen-go" \
		--go-grpc_out=$(OUT_PATH) --go-grpc_opt=paths=source_relative --plugin protoc-gen-go-grpc="${GOBIN}/protoc-gen-go-grpc" \
		--validate_out="lang=go,paths=source_relative:$(OUT_PATH)" --plugin protoc-gen-validate=$(LOCAL_BIN)/protoc-gen-validate \
		--grpc-gateway_out=$(OUT_PATH) --grpc-gateway_opt=paths=source_relative --plugin protoc-gen-grpc-gateway=$(LOCAL_BIN)/protoc-gen-grpc-gateway \
		--openapiv2_out=$(OUT_PATH) --plugin=protoc-gen-openapiv2=$(LOCAL_BIN)/protoc-gen-openapiv2 \
		api/orders/contract.proto 
	go mod tidy
