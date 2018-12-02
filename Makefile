BIN     := bin/
CMD     := cmd/
OS      := linux
ARCH    := amd64
ADDED   := -linkmode external -extldflags '$(LDFLAGS)'
TARGET  := pie
SRC     := $(shell find $(CMD) -type f -name "*.go")
VERSION := DEVELOP
LINUX   := linux
TARGETS := $(LINUX)
FLAGS   := -ldflags '$(ADDED) -s -w -X main.vers=$(VERSION)' -gcflags=all=-trimpath=$(GOPATH) -asmflags=all=-trimpath=$(GOPATH)
OUTPUT  := $(BIN)$(OS)/$(ARCH)/
GOBUILD := GOOS=$(OS) GOARCH=$(ARCH) go build -o $(OUTPUT)
GOFLAGS := $(FLAGS) -buildmode=$(TARGET) $(CMD)common.go $(CMD)
APPS    := survey stitcher
RSRC    := /usr/share/survey/resources
TMPL    := templates/
SYSD    := /lib/systemd/system/
TMPD    := /usr/lib/tmpfiles.d/
ETC     := /etc/survey/

all: clean build format

build: $(TARGETS)

target: $(APPS)

$(APPS):
	mkdir -p $(OUTPUT)
	$(GOBUILD)$@ $(GOFLAGS)$@.go

windows:
	make target OS=windows TARGET=exe ADDED=''

$(LINUX):
	make target

arm8:
	make target TARGET=exe ARCH=arm64 ADDED=''

clean:
	rm -rf $(BIN)
	mkdir -p $(BIN)

format:
	exit $(shell goimports -l $(SRC) | wc -l)

install:
	install -Dm 755 -d $(DESTDIR)$(ETC)
	install -Dm 644 supporting/example.config $(DESTDIR)$(ETC)
	install -Dm 644 supporting/settings.conf $(DESTDIR)$(ETC)
	install -Dm 755 $(BIN)/linux/amd64/survey $(DESTDIR)/usr/bin/survey
	install -Dm 755 $(BIN)/linux/amd64/stitcher $(DESTDIR)usr/bin/survey-stitcher
	install -Dm 755 -d $(DESTDIR)(SYSD)
	install -Dm 644 supporting/survey.service $(DESTDIR)$(SYSD)
	install -Dm 755 -d $(DESTDIR)$(TMPD)
	install -Dm 644 supporting/tmpfiles.d $(DESTDIR)$(TMPD)survey.conf
	for f in $(shell find $(TMPL) -type d | cut -d "/" -f 2-); do install -Dm755 -d $(DESTDIR)$(RSRC)/$$f; done
	for f in $(shell find $(TMPL) -type f | cut -d "/" -f 2-); do install -Dm755 $(TMPL)/$$f $(DESTDIR)$(RSRC)/$$f; done
