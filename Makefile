BIN_NAME := umeed-indexer
OUT_BUILD_PATH := $(CURDIR)/build
MAIN_FILE := $(CURDIR)/main.go


build:
	go build -o $(OUT_BUILD_PATH)/$(BIN_NAME) $(MAIN_FILE)

## Indexer
run:
	go run main.go start

# Generate GraphQL code
generate:
	@echo "Generating GraphQL code..."
	@go run github.com/99designs/gqlgen@v0.17.42 generate

## Database
run-firestore:
	./scripts/firestore-emulator-run.sh

clean:
	rm -rf $(OUT_BUILD_PATH)/$(BIN_NAME) $(MAIN_FILE)

.PHONY: build clean