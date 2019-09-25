VERSION ?= master
FLAGS   := -ldflags '-linkmode external -extldflags $(LDFLAGS) -s -w -X main.vers=$(VERSION)' -gcflags=all=-trimpath=$(GOPATH) -asmflags=all=-trimpath=$(GOPATH) -buildmode=pie
TMPL    := $(shell find templates/ -type f)
OBJECTS := survey survey-stitcher
BINDATA := core/bindata.go

.PHONY: tests clean all lint

all: $(OBJECTS) lint tests

$(BINDATA): $(TMPL)
	go-bindata -o $(BINDATA) -pkg core $(TMPL)

$(OBJECTS): $(BINDATA) $(shell find . -type f -name "*.go")
	go build -o $@ $(FLAGS) cmd/$@.go

tests: $(BINARY)
	cd test/ && ./run.sh

clean:
	rm -f $(BINDATA) $(OBJECTS)

lint:
	@golinter
