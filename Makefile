SERVER_BIN := bin/pirate-server
SERVER_SRC := cmd/pirate-server/main.go $(shell find pirate -type f -name '*.go')
CONFIG     := config.yml
DOCKERFILE := Dockerfile

.PHONY: all run test bench docker clean

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

docker: $(CONFIG)
	@echo "[build] docker"
	@docker build -t innogames/pirate:latest .

clean:
	@echo "[clean] cleaning binaries"
	@rm -rf $(dir $(SERVER_BIN))
