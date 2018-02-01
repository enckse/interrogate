DIST=dist/

all: clean win linux

makedepends:
	pacman -S electron npm wine

depends:
	npm install

clean:
	rm -rf $(DIST)
	mkdir -p $(DIST)

win:
	npm run windows

linux:
	npm run linux
