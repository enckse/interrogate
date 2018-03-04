BIN=bin/
SRC=$(shell find src/ -type f | grep "\.go$$")
CMD=go build -o $(BIN)survey $(SRC)

build-objects = mkdir -p $(BIN)$1/$2; \
				GOOS=$1 GOARCH=$2 go build -o $(BIN)$1/$2/survey $(SRC)

all: clean build

build: linux arm8 windows

windows:
	$(call build-objects,windows,amd64)

linux:
	$(call build-objects,linux,amd64)

arm8:
	$(call build-objects,linux,arm64)

clean:
	rm -rf $(BIN)
	mkdir -p $(BIN)

format:
	exit $(shell gofmt -l $(SRC) | wc -l)

dependencies:
	git submodule update --init
	curl https://code.jquery.com/jquery-3.2.1.min.js > templates/static/jquery.min.js
