SERVER_BIN := bin/pirate-server
SERVER_SRC := cmd/pirate-server/main.go $(shell find pirate -type f -name '*.go')
CONFIG     := config.yml
DOCKERFILE := Dockerfile

.PHONY: all
all: $(SERVER_BIN)

$(SERVER_BIN): $(SERVER_SRC)
	@echo "+ $@"
	go build -o $@ $<

.PHONY: run
run: $(SERVER_BIN) $(CONFIG)
	@echo "+ $@"
	$(SERVER_BIN) -config $(CONFIG)

$(CONFIG):
	@echo "+ $@"
	cp example-config.yml $@

.PHONY: test
test:
	@echo "+ $@"
	go test ./pirate/...

.PHONY: bench
bench:
	@echo "+ $@"
	go test ./pirate/... -bench=.

.PHONY: docker
docker: $(CONFIG)
	@echo "+ $@"
	docker build -t innogames/pirate:latest .

.PHONY: clean
clean:
	@echo "+ $@"
	rm -rf $(dir $(SERVER_BIN))
