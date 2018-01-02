FILES=$(shell find . -type f | grep "\.py")

dependencies:
	git submodule update --init
	curl https://code.jquery.com/jquery-3.2.1.min.js > survey/static/jquery.min.js

install: dependencies
	pip install flask

analyze:
	pip install pep257 pycodestyle
	pycodestyle $(FILES)
	pep257 $(FILES)
