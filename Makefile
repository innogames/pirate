.PHONY: all clean

all: server

server:
	go build -o bin/pirate-server cmd/pirate-server/main.go

server\:linux: init
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -ldflags '-s' -o bin/pirate-server cmd/pirate-server/main.go

server\:run: server init
	./bin/pirate-server -config config.yml

init:
	@if ! [ -a config.yml ]; then \
		echo "Copying example-config.yml to config.yml"; \
		cp example-config.yml config.yml; \
	fi

test:
	go test ./pirate/...

bench:
    go test ./pirate/... -bench=.