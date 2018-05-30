BIN     := bin/
SRC     := $(shell find cmd/ -type f -name "*.go")
VERS    := master
LINUX   := linux
ARM8    := arm8
WINDOWS := windows
TARGETS := $(LINUX) $(ARM8) $(WINDOWS)
FLAGS   := -ldflags '-s -w -X main.vers=$(VERS)'

build-objects =	GOOS=$1 GOARCH=$2 go build -o $(BIN)$1/$2/survey $(FLAGS) $(SRC)

all: clean build format

build: $(TARGETS)

$(WINDOWS):
	$(call build-objects,windows,amd64)

$(LINUX):
	$(call build-objects,linux,amd64)

$(ARM8):
	$(call build-objects,linux,arm64)

clean:
	rm -rf $(BIN)
	mkdir -p $(BIN)

format:
	exit $(shell goimports -l $(SRC) | wc -l)
