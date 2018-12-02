BIN     := bin/
CMD     := cmd/
SRC     := $(shell find $(CMD) -type f -name "*.go")
VERSION := DEVELOP
FLAGS   := -ldflags '-linkmode external -extldflags $(LDFLAGS) -s -w -X main.vers=$(VERSION)' -gcflags=all=-trimpath=$(GOPATH) -asmflags=all=-trimpath=$(GOPATH) -buildmode=pie $(CMD)common.go $(CMD)
APPS    := survey stitcher
RSRC    := /usr/share/survey/resources
TMPL    := templates/
SYSD    := /lib/systemd/system/
TMPD    := /usr/lib/tmpfiles.d/
ETC     := /etc/survey/

all: clean build format

build: $(APPS)

$(APPS):
	go build -o $(BIN)$@ $(FLAGS)$@.go

clean:
	rm -rf $(BIN)
	mkdir -p $(BIN)

format:
	exit $(shell goimports -l $(SRC) | wc -l)

install:
	install -Dm 755 -d $(DESTDIR)$(ETC)
	install -Dm 644 supporting/example.config $(DESTDIR)$(ETC)
	install -Dm 644 supporting/settings.conf $(DESTDIR)$(ETC)
	install -Dm 755 $(BIN)survey $(DESTDIR)/usr/bin/survey
	install -Dm 755 $(BIN)stitcher $(DESTDIR)usr/bin/survey-stitcher
	install -Dm 755 -d $(DESTDIR)(SYSD)
	install -Dm 644 supporting/survey.service $(DESTDIR)$(SYSD)
	install -Dm 755 -d $(DESTDIR)$(TMPD)
	install -Dm 644 supporting/tmpfiles.d $(DESTDIR)$(TMPD)survey.conf
	for f in $(shell find $(TMPL) -type d | cut -d "/" -f 2-); do install -Dm755 -d $(DESTDIR)$(RSRC)/$$f; done
	for f in $(shell find $(TMPL) -type f | cut -d "/" -f 2-); do install -Dm755 $(TMPL)/$$f $(DESTDIR)$(RSRC)/$$f; done
