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

build-objects =	GOOS=$1 GOARCH=$2 go build -o $(BIN)$1/$2/survey $(FLAGS) -buildmode=$3 $(OBJS) $(CMD)common_$1_$2.go $(CMD)survey.go

all: clean build format

build: $(TARGETS)

$(WINDOWS):
	$(call build-objects,windows,amd64,exe)

$(LINUX):
	$(call build-objects,linux,amd64,pie)
	go build -o $(BIN)stitcher $(FLAGS) -buildmode=pie $(OBJS) $(CMD)stitcher.go $(CMD)common_linux_amd64.go

$(ARM8):
	$(call build-objects,linux,arm64,exe)

clean:
	rm -rf $(BIN)
	mkdir -p $(BIN)

format:
	exit $(shell goimports -l $(SRC) | wc -l)
