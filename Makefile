BIN     := bin/
VERSION ?= master
FLAGS   := -ldflags '-linkmode external -extldflags $(LDFLAGS) -s -w -X main.vers=$(VERSION)' -gcflags=all=-trimpath=$(GOPATH) -asmflags=all=-trimpath=$(GOPATH) -buildmode=pie
TMPL    := $(shell find templates/ -type f)
FORMAT  := $(BIN)format
BINARY  := $(BIN)survey
SRC     := $(shell find . -type f -name "*.go")
STITCH  := survey-stitcher
PY      := $(BIN)$(STITCH)

all: $(BINARY) $(FORMAT) $(PY) tests

bindata.go: $(TMPL)
	go-bindata $(TMPL)

$(BINARY): bindata.go $(SRC)
	go build -o $(BIN)survey $(FLAGS) *.go

$(PY): $(STITCH)
	install -Dm755 $(STITCH) $(PY)

tests:
	cd test/ && ./run.sh

clean:
	rm -f bindata.go
	rm -rf $(BIN)
	mkdir -p $(BIN)

$(FORMAT): $(SRC) $(STITCH)
	goformatter
	pycodestyle $(STITCH)
	pydocstyle $(STITCH)
	@touch $(FORMAT)
