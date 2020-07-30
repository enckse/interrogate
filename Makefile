VERSION ?= master
FLAGS   := -ldflags '-linkmode external -extldflags $(LDFLAGS) -s -w -X main.vers=$(VERSION)' -trimpath -buildmode=pie -mod=readonly -modcacherw
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

deb:
ifeq ($(VERSION),master)
	$(error VERSION can NOT be master)
endif
	podman build --tag debian:survey-deb -f ./dockerfiles/build/debian/Dockerfile --volume $(PWD):/debs --build-arg SURVEY_VERSION=$(VERSION)
