dependencies:
	git submodule update --init
	curl https://code.jquery.com/jquery-3.2.1.min.js > survey/static/jquery.min.js

install: dependencies
	pip install flask
	mkdir -p artifacts

analyze:
	pip install pep257
	pip install pep8
	pep8 *.py
	pep257 *.py
