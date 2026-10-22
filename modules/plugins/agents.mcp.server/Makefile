NAME := $(notdir $(patsubst %/,%,$(CURDIR)))
EXE ?=
ifeq ($(OS),Windows_NT)
EXE := .exe
endif

.PHONY: all build backend_build frontend_build lint test clean

all: build

build: backend_build frontend_build

backend_build:
	@if [ -d backend ]; then \
		mkdir -p bin && \
		go build -o bin/$(NAME)$(EXE) ./backend ; \
	fi

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
	go vet ./...
	golangci-lint run ./...
	@if [ -f package.json ]; then npm run lint --if-present; fi

test:
	go test ./...
	@if [ -f package.json ]; then npm test; fi

clean:
	rm -rf bin dist
