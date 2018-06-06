BIN     := bin/
CMD     := cmd/
OBJS    := $(CMD)objects.go
SRC     := $(shell find $(CMD) -type f -name "*.go")
VERS    := $(shell git describe --long | sed "s/\([^-]*-g\)/r\1/;s/-/./g")
LINUX   := linux
ARM8    := arm8
WINDOWS := windows
TARGETS := $(LINUX) $(ARM8) $(WINDOWS)
FLAGS   := -ldflags '-s -w -X main.vers=$(VERS)'

build-objects = GOOS=$1 GOARCH=$2 go build -o $(BIN)survey-$1-$2 $(FLAGS) -buildmode=$3 $(OBJS) $(CMD)common_$1_$2.go $(CMD)survey.go
build-stitcher = GOOS=$1 GOARCH=$2 go build -o $(BIN)stitcher-$1-$2 $(FLAGS) -buildmode=$3 $(OBJS) $(CMD)common_$1_$2.go $(CMD)stitcher.go
build-all = $(call build-objects,$1,$2,$3) && $(call build-stitcher,$1,$2,$3)

all: clean build format

build: $(TARGETS)

$(WINDOWS):
	$(call build-objects,windows,amd64,exe)

$(LINUX):
	$(call build-all,linux,amd64,pie)

$(ARM8):
	$(call build-all,linux,arm64,exe)

clean:
	rm -rf $(BIN)
	mkdir -p $(BIN)

format:
	exit $(shell goimports -l $(SRC) | wc -l)
