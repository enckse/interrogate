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
OUTPUT  := $(BIN)$(OS)/$(ARCH)/
GOBUILD := GOOS=$(OS) GOARCH=$(ARCH) go build -o $(OUTPUT)
GOFLAGS := $(FLAGS) -buildmode=$(TARGET) $(CMD)common.go $(CMD)
APPS    := survey stitcher
RSRC    := usr/share/survey/resources
TMPL    := templates/
SYSD    := lib/systemd/system/

all: clean build format

build: $(TARGETS)

target: $(APPS)

$(APPS):
	mkdir -p $(OUTPUT)
	$(GOBUILD)$@ $(GOFLAGS)$@.go

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
	exit $(shell gofmt -l $(SRC) | wc -l)

install:
	install -Dm 755 -d $(DESTDIR)etc/survey
	install -Dm 644 supporting/example.config $(DESTDIR)etc/survey/
	install -Dm 644 supporting/settings.conf $(DESTDIR)etc/survey/
	install -Dm 755 $(BIN)/linux/amd64/survey $(DESTDIR)usr/bin/survey
	install -Dm 755 $(BIN)/linux/amd64/stitcher $(DESTDIR)usr/bin/survey-stitcher
	install -Dm 755 -d $(SYSD)
	install -Dm 644 supporting/survey.service $(DESTDIR)lib/systemd/system/
	for f in $(shell find $(TMPL) -type d | cut -d "/" -f 2-); do install -Dm755 -d $(DESTDIR)$(RSRC)/$$f; done
	for f in $(shell find $(TMPL) -type f | cut -d "/" -f 2-); do install -Dm755 $(TMPL)/$$f $(DESTDIR)$(RSRC)/$$f; done
	install -Dm 755 -d $(DESTDIR)var/cache/survey
	install -Dm 755 -d $(DESTDIR)var/tmp/survey
