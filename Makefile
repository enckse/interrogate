VERSION ?= master
FLAGS   := -ldflags '-linkmode external -extldflags "-zrelro -znow $(LDFLAGS)" -s -w -X main.vers=$(VERSION)' -gcflags=all=-trimpath=$(PWD) -asmflags=all=-trimpath=$(PWD) -buildmode=pie
TMPL    := $(shell find templates/ -type f)
OBJECTS := survey survey-stitcher
BINDATA := internal/bindata.go

.PHONY: tests clean all lint

all: $(OBJECTS) lint tests

$(BINDATA): $(TMPL)
	go-bindata -o $(BINDATA) -pkg internal $(TMPL)

$(OBJECTS): $(BINDATA) $(shell find . -type f -name "*.go")
	go build -o $@ $(FLAGS) cmd/$@/main.go

tests: $(BINARY)
	cd test/ && ./run.sh

clean:
	rm -f $(BINDATA) $(OBJECTS)

lint:
	@golinter
