BIN     := bin/
CMD     := cmd/
OBJS    := $(CMD)common.go
SRC     := $(shell find $(CMD) -type f -name "*.go")
VERSION ?= $(shell git describe --long | sed "s/\([^-]*-g\)/r\1/;s/-/./g")
LINUX   := linux
ARM8    := arm8
WINDOWS := windows
TARGETS := $(LINUX) $(ARM8) $(WINDOWS)
FLAGS   := -ldflags '-s -w -X main.vers=$(VERSION)'

build-object = GOOS=$1 GOARCH=$2 go build -o $(BIN)$4-$1-$2 $(FLAGS) -buildmode=$3 $(OBJS) $(CMD)$4.go
build-survey = $(call build-object,$1,$2,$3,survey)
build-stitcher = $(call build-object,$1,$2,$3,stitcher)
build-all = $(call build-survey,$1,$2,$3) && $(call build-stitcher,$1,$2,$3)

all: clean build format

build: $(TARGETS)

$(WINDOWS):
	$(call build-survey,windows,amd64,exe)

$(LINUX):
	$(call build-all,linux,amd64,pie)

$(ARM8):
	$(call build-all,linux,arm64,exe)

clean:
	rm -rf $(BIN)
	mkdir -p $(BIN)

format:
	exit $(shell goimports -l $(SRC) | wc -l)
