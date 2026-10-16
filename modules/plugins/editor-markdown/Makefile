NAME := $(notdir $(patsubst %/,%,$(CURDIR)))

.PHONY: all build frontend_build lint test clean

all: build

build: frontend_build

frontend_build:
	@if [ -f package.json ]; then \
		npm run build ; \
	elif [ -d frontend ]; then \
		mkdir -p dist ; \
		if [ -f frontend/styles.css ]; then \
			cp frontend/styles.css dist/styles.css ; \
		fi ; \
	fi

lint:
	@if [ -f package.json ]; then npm run lint --if-present; fi

test:
	@if [ -f package.json ]; then npm test; fi

clean:
	rm -rf bin dist
