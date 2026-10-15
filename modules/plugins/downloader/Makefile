NAME := $(notdir $(patsubst %/,%,$(CURDIR)))
EXE ?=
ifeq ($(OS),Windows_NT)
EXE := .exe
endif

.PHONY: all build backend_build lint test clean

all: build

build: backend_build

backend_build:
	@if [ -d backend ]; then \
		mkdir -p bin && \
		go build -o bin/$(NAME)$(EXE) ./backend ; \
	fi

lint:
	go vet ./...
	golangci-lint run ./...

test:
	go test ./...

clean:
	rm -rf bin dist
