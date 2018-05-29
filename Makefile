BIN=bin/
SRC=$(shell find cmd/ -type f | grep "\.go$$")
CMD=go build -o $(BIN)survey $(SRC)
VERS=
ifeq ($(VERS),)
	VERS=master
endif

build-objects = mkdir -p $(BIN)$1/$2 || exit 1; \
				GOOS=$1 GOARCH=$2 go build -o $(BIN)$1/$2/survey -ldflags '-X main.vers=$(VERS)' $(SRC)

all: clean build format

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
	exit $(shell goimports -l $(SRC) | wc -l)
