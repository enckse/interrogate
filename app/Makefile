DIST=dist/

all: clean win linux

makedepends:
	pacman -S electron npm

depends:
	npm install
	npm install electron-builder

clean:
	rm -rf $(DIST)
	mkdir -p $(DIST)

windepends:
	pacman -S wine

win:
	npm run windows

linux:
	npm run linux
