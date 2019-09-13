BIN     := bin/
VERSION := $(BUILD_VERSION)
ifeq ($(VERSION),)
       VERSION := DEVELOP
endif
FLAGS   := -ldflags '-linkmode external -extldflags $(LDFLAGS) -s -w -X main.vers=$(VERSION)' -gcflags=all=-trimpath=$(GOPATH) -asmflags=all=-trimpath=$(GOPATH) -buildmode=pie
TMPL    := $(shell find templates/ -type f)
SYSD    := /lib/systemd/system/
TMPD    := /usr/lib/tmpfiles.d/
ETC     := /etc/survey/
SUPPORT := supporting/

all: clean build format

build:
	go-bindata $(TMPL)
	go build -o $(BIN)survey $(FLAGS) survey.go bindata.go

clean:
	rm -rf $(BIN)
	mkdir -p $(BIN)

format:
	goformatter

install:
	install -Dm 755 -d $(DESTDIR)$(ETC)
	install -Dm 644 $(SUPPORT)example.json $(DESTDIR)$(ETC)
	install -Dm 644 $(SUPPORT)settings.conf $(DESTDIR)$(ETC)
	install -Dm 755 $(BIN)survey $(DESTDIR)/usr/bin/survey
	install -Dm 755 -d $(DESTDIR)$(SYSD)
	install -Dm 644 $(SUPPORT)survey.service $(DESTDIR)$(SYSD)
	install -Dm 755 -d $(DESTDIR)$(TMPD)
	install -Dm 644 $(SUPPORT)tmpfiles.d $(DESTDIR)$(TMPD)survey.conf
	install -Dm 755 $(SUPPORT)stitcher.py $(DESTDIR)/usr/bin/survey-stitcher
	install -Dm 755 -d $(DESTDIR)/usr/share/survey/resources
