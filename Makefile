BIN     := bin/
CMD     := cmd/
OS      := linux
ARCH    := amd64
ADDED   := -linkmode external -extldflags '$(LDFLAGS)'
TRIMS   := -gcflags=all=-trimpath=$(GOPATH) -asmflags=all=-trimpath=$(GOPATH)
TARGET  := pie
SRC     := $(shell find $(CMD) -type f -name "*.go")
VERSION ?= $(shell git describe --long | sed "s/\([^-]*-g\)/r\1/;s/-/./g")
LINUX   := linux
ARM8    := arm8
WINDOWS := windows
TARGETS := $(LINUX) $(ARM8) $(WINDOWS)
FLAGS   := -ldflags '$(ADDED) -s -w -X main.vers=$(VERSION)' $(TRIMS)
GOBUILD := GOOS=$(OS) GOARCH=$(ARCH) go build -o $(BIN)
GOFLAGS := $(FLAGS) -buildmode=$(TARGET) $(CMD)common.go $(CMD)
APPS    := survey stitcher

all: clean build format

build: $(TARGETS)

target: $(APPS)

$(APPS):
	$(GOBUILD)$@-$(OS)-$(ARCH) $(GOFLAGS)$@.go

$(WINDOWS):
	make target OS=windows TARGET=exe ADDED='' TRIM=''

$(LINUX):
	make target

$(ARM8):
	make target TARGET=exe ARCH=arm64 ADDED='' TRIM=''

clean:
	rm -rf $(BIN)
	mkdir -p $(BIN)

format:
	exit $(shell goimports -l $(SRC) | wc -l)
