QUESTIONS="questions"
EXAMPLE="^(example|cont)$\.config"
EXAMPLES=$(shell ls $(QUESTIONS) | grep -E $(EXAMPLE) | sort -r)
DEFINITIONS=$(shell ls $(QUESTIONS) | grep -E -v $(EXAMPLE) | sort)
OUTPUT=disk
.PHONY: all

define run
	python survey.py --port 8080 --questions $(shell echo $2 | sed "s/.config//g") --output $1
endef

all: 
	$(call run,$(OUTPUT),$(DEFINITIONS))

examples:
	$(call run,off,$(EXAMPLES))

dependencies:
	git submodule update --init
	curl https://code.jquery.com/jquery-3.1.1.min.js > static/jquery.min.js

install: dependencies
	pip install flask
	mkdir -p artifacts

analyze:
	pip install pep257
	pip install pep8
	pep8 *.py
	pep257 *.py

