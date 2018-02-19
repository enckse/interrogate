BIN=bin/
SRC=$(shell find survey/ -type f | grep "\.go$$")

all: clean build

build:
	go build -o $(BIN)survey $(SRC)

clean:
	rm -rf $(BIN)
	mkdir -p $(BIN)

format:
	exit $(shell gofmt -l $(SRC) | wc -l)

dependencies:
	curl https://code.jquery.com/jquery-3.2.1.min.js > survey/static/jquery.min.js
