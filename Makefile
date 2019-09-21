BIN     := bin/
VERSION ?= master
FLAGS   := -ldflags '-linkmode external -extldflags $(LDFLAGS) -s -w -X main.vers=$(VERSION)' -gcflags=all=-trimpath=$(GOPATH) -asmflags=all=-trimpath=$(GOPATH) -buildmode=pie
TMPL    := $(shell find templates/ -type f)
FORMAT  := $(BIN)format
BINARY  := $(BIN)survey $(BIN)survey-stitcher
SRC     := $(shell find . -type f -name "*.go")
STITCH  := survey-stitcher
PY      := $(BIN)$(STITCH)
BINDATA := core/bindata.go

all: $(BINARY) $(FORMAT) $(PY) tests

$(BINDATA): $(TMPL)
	go-bindata -o $(BINDATA) -pkg core $(TMPL)

$(BINARY): $(BINDATA) $(SRC)
	go build -o $@ $(FLAGS)  $(shell echo $@ | cut -d "/" -f 2).go

tests: $(BINARY)
	cd test/ && ./run.sh

clean:
	rm -f $(BINDATA)
	rm -rf $(BIN)
	mkdir -p $(BIN)

$(FORMAT): $(SRC) $(STITCH)
	goformatter
	pycodestyle $(STITCH)
	pydocstyle $(STITCH)
	@touch $(FORMAT)
