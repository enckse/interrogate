DIST=dist/

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

win: clean
	npm run windows

linux: clean
	npm run linux
