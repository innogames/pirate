SERVER_BIN := bin/pirate-server
SERVER_SRC := cmd/pirate-server/main.go $(shell find pirate -type f -name '*.go')
CONFIG     := config.yml

.PHONY: all test bench clean

all: $(SERVER_BIN)

$(SERVER_BIN): $(SERVER_SRC)
	@echo "[compile] $@"
	@go build -o $@ $<

run: $(SERVER_BIN) $(CONFIG)
	@echo "[run] server"
	@$(SERVER_BIN) -config $(CONFIG)

$(CONFIG):
	@echo "[init] config.yml"
	@cp example-config.yml $@

test:
	@echo "[test] running tests"
	@go test ./pirate/...

bench:
	@echo "[test] running benchmarks"
	@go test ./pirate/... -bench=.

clean:
	@echo "[clean] cleaning binaries"
	@rm -rf $(dir $(SERVER_BIN))
