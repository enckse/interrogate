DIST=dist/

all: clean win linux

makedepends:
	pacman -S electron npm wine

depends:
	npm install
	npm install electron-builder

clean:
	rm -rf $(DIST)
	mkdir -p $(DIST)

win:
	npm run windows

linux:
	npm run linux
