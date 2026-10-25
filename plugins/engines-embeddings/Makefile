# The four targets every plugin answers, whatever it is made of: a backend in Go, a frontend built
# by Vite, or both.
NAME := $(notdir $(patsubst %/,%,$(CURDIR)))
EXE ?=
ifeq ($(OS),Windows_NT)
EXE := .exe
endif

.PHONY: build lint test clean

build:
	@if [ -d backend ]; then mkdir -p bin && go build -ldflags="-s -w" -o bin/$(NAME)$(EXE) ./backend; fi
	@if [ -f package.json ]; then npm run build; fi

lint:
	@if [ -f go.mod ]; then go vet ./... && golangci-lint run ./...; fi
	@if [ -f package.json ]; then npm run lint && npm run typecheck; fi

test:
	@if [ -f go.mod ]; then go test ./...; fi
	@if [ -f package.json ]; then npm test; fi

clean:
	rm -rf bin dist
