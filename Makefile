BIN=bin/
SRC=$(shell find src/ -type f | grep "\.go$$")
CMD=go build -o $(BIN)survey $(SRC)

all: clean build

build: arm8 linux windows

windows:
	GOOS=windows GOARCH=amd64 $(CMD)

linux:
	GOOS=linux GOARCH=amd64 $(CMD)

arm8:
	GOOS=linux GOARCH=arm64 $(CMD)

clean:
	rm -rf $(BIN)
	mkdir -p $(BIN)

format:
	exit $(shell gofmt -l $(SRC) | wc -l)

dependencies:
	curl https://code.jquery.com/jquery-3.2.1.min.js > templates/static/jquery.min.js
