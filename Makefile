SERVER_BIN := bin/pirate-server
SERVER_SRC := cmd/pirate-server/main.go $(shell find pirate -type f -name '*.go')
CONFIG     := config.yml
DOCKERFILE := Dockerfile

.PHONY: all
all: $(SERVER_BIN)

$(SERVER_BIN): $(SERVER_SRC)
	@echo "[compile] $@"
	@go build -o $@ $<

.PHONY: run
run: $(SERVER_BIN) $(CONFIG)
	@echo "[run] server"
	@$(SERVER_BIN) -config $(CONFIG)

$(CONFIG):
	@echo "[init] config.yml"
	@cp example-config.yml $@

.PHONY: test
test:
	@echo "[test] running tests"
	@go test ./pirate/...

.PHONY: bench
bench:
	@echo "[test] running benchmarks"
	@go test ./pirate/... -bench=.

.PHONY: docker
docker: $(CONFIG)
	@echo "[build] docker"
	@docker build -t innogames/pirate:latest .

.PHONY: clean
clean:
	@echo "[clean] cleaning binaries"
	@rm -rf $(dir $(SERVER_BIN))
