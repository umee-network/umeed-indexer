BIN_NAME := umeed-indexer
OUT_BUILD_PATH := $(CURDIR)/build
MAIN_FILE := $(CURDIR)/main.go


build:
	go build -o $(OUT_BUILD_PATH)/$(BIN_NAME) $(MAIN_FILE)

## Indexer
run:
	go run main.go start

## Database
run-firestore:
	./scripts/firestore-emulator-run.sh
