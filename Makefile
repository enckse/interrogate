VERSION ?= master
FLAGS   := -ldflags '-linkmode external -extldflags $(LDFLAGS) -s -w -X main.vers=$(VERSION)' -trimpath -buildmode=pie -mod=readonly -modcacherw
TMPL    := $(shell find templates/ -type f)
OBJECTS := interrogate interrogate-stitcher
BINDATA := internal/bindata.go
DESTDIR :=

.PHONY: tests clean all

all: build tests
	
build: $(OBJECTS)

$(BINDATA): $(TMPL)
	go-bindata -o $(BINDATA) -pkg internal $(TMPL)

$(OBJECTS): $(BINDATA) $(shell find . -type f -name "*.go")
	go build -o $@ $(FLAGS) cmd/$@/main.go

tests: $(BINARY)
	cd test/ && ./run.sh

clean:
	rm -f $(BINDATA) $(OBJECTS)

install: build
	install -d $(DESTDIR)/etc/survey
	install -Dm755 interrogate $(DESTDIR)/usr/bin/
	install -Dm755 interrogate-stitcher $(DESTDIR)/usr/bin/
	install -Dm644 configs/example.yaml $(DESTDIR)/etc/interrogate/
	install -Dm644 configs/settings.conf $(DESTDIR)/etc/interrogate/
	install -Dm644 configs/systemd/interrogate.conf $(DESTDIR)/usr/lib/tmpfiles.d/
	install -Dm644 configs/systemd/interrogate.service $(DESTDIR)/usr/lib/systemd/system/
