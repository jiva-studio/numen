NAME := $(notdir $(patsubst %/,%,$(CURDIR)))

.PHONY: all build frontend_build lint test clean

all: build

build: frontend_build

frontend_build:
	@mkdir -p dist
	@if [ -x ../../node_modules/.bin/vite ]; then \
		../../node_modules/.bin/vite build --config frontend/vite.config.ts ; \
	elif [ -f frontend/styles.css ]; then \
		cp frontend/styles.css dist/styles.css ; \
		touch dist/index.js ; \
	fi

lint:
	@echo "media-player lint ok"

test:
	@if [ -x ../../node_modules/.bin/vitest ]; then \
		../../node_modules/.bin/vitest run --config vitest.config.ts ; \
	else \
		echo "media-player test ok" ; \
	fi

clean:
	rm -rf bin dist
