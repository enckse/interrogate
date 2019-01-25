BIN     := bin/
VERSION := DEVELOP
FLAGS   := -ldflags '-linkmode external -extldflags $(LDFLAGS) -s -w -X main.vers=$(VERSION)' -gcflags=all=-trimpath=$(GOPATH) -asmflags=all=-trimpath=$(GOPATH) -buildmode=pie
RSRC    := /usr/share/survey/resources
TMPL    := templates/
SYSD    := /lib/systemd/system/
TMPD    := /usr/lib/tmpfiles.d/
ETC     := /etc/survey/
SUPPORT := supporting/

all: clean build format

build:
	go build -o $(BIN)survey $(FLAGS) survey.go

clean:
	rm -rf $(BIN)
	mkdir -p $(BIN)

format:
	exit $(shell goimports -l survey.go | wc -l)

install:
	install -Dm 755 -d $(DESTDIR)$(ETC)
	install -Dm 644 $(SUPPORT)example.config $(DESTDIR)$(ETC)
	install -Dm 644 $(SUPPORT)settings.conf $(DESTDIR)$(ETC)
	install -Dm 755 $(BIN)survey $(DESTDIR)/usr/bin/survey
	install -Dm 755 -d $(DESTDIR)$(SYSD)
	install -Dm 644 $(SUPPORT)survey.service $(DESTDIR)$(SYSD)
	install -Dm 755 -d $(DESTDIR)$(TMPD)
	install -Dm 644 $(SUPPORT)tmpfiles.d $(DESTDIR)$(TMPD)survey.conf
	install -Dm 755 $(SUPPORT)stitcher.py $(DESTDIR)/usr/bin/survey-stitcher
	for f in $(shell find $(TMPL) -type d | cut -d "/" -f 2-); do install -Dm755 -d $(DESTDIR)$(RSRC)/$$f; done
	for f in $(shell find $(TMPL) -type f | cut -d "/" -f 2-); do install -Dm644 $(TMPL)/$$f $(DESTDIR)$(RSRC)/$$f; done
