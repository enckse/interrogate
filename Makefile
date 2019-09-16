BIN     := bin/
VERSION ?= master
FLAGS   := -ldflags '-linkmode external -extldflags $(LDFLAGS) -s -w -X main.vers=$(VERSION)' -gcflags=all=-trimpath=$(GOPATH) -asmflags=all=-trimpath=$(GOPATH) -buildmode=pie
TMPL    := $(shell find templates/ -type f)
FORMAT  := $(BIN)format
BINARY  := $(BIN)survey
SRC     := $(shell find . -type f -name "*.go")

all: $(BINARY) $(FORMAT)

bindata.go: $(TMPL)
	go-bindata $(TMPL)

$(BINARY): bindata.go $(SRC)
	go build -o $(BIN)survey $(FLAGS) *.go

clean:
	rm -f bindata.go
	rm -rf $(BIN)
	mkdir -p $(BIN)

$(FORMAT): $(SRC)
	goformatter
	@touch $(FORMAT)
