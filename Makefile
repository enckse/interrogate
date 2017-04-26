.PHONY: all

all: 
	python survey.py --port 8080 --questions example cont --output off

install:
	pip install flask
	mkdir -p artifacts
	git submodule update
	curl https://code.jquery.com/jquery-3.1.1.min.js > static/jquery.min.js

analyze:
	pip install pep257
	pip install pep8
	pep8 *.py
	pep257 *.py

