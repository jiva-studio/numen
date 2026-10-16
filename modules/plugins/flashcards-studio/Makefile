NAME := $(notdir $(patsubst %/,%,$(CURDIR)))

.PHONY: all build frontend_build lint test clean

all: build

build: frontend_build

frontend_build:
	@mkdir -p dist
	@if [ -d frontend ]; then \
		if [ -f frontend/styles.css ]; then \
			cp frontend/styles.css dist/styles.css ; \
		fi ; \
	fi

lint:
	@if [ -f frontend/package.json ]; then cd frontend && npm run lint 2>/dev/null || true; fi

test:
	@if [ -f frontend/package.json ]; then cd frontend && npm test 2>/dev/null || true; fi

clean:
	rm -rf bin dist
