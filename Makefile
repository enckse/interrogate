VERSION ?= master
FLAGS   := -ldflags '-linkmode external -extldflags $(LDFLAGS) -s -w -X main.vers=$(VERSION)' -gcflags=all=-trimpath=$(GOPATH) -asmflags=all=-trimpath=$(GOPATH) -buildmode=pie
TMPL    := $(shell find templates/ -type f)
OBJECTS := survey survey-stitcher
SRC     := $(shell find . -type f -name "*.go")
BINDATA := core/bindata.go

.PHONY: tests lint clean all

all: $(OBJECTS) lint tests

$(BINDATA): $(TMPL)
	go-bindata -o $(BINDATA) -pkg core $(TMPL)

$(OBJECTS): $(BINDATA) $(SRC)
	go build -o $@ $(FLAGS) cmd/$@.go 

tests: $(OBJECTS)
	cd test/ && ./run.sh

clean:
	rm -f $(BINDATA)
	rm -f $(OBJECTS)

lint:
	@golinter
