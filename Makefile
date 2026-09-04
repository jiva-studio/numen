# Handles, not a build system.
#
# Each module builds itself with its own tools — Go with go, the interface with
# npm. What is here is the list of things worth doing and where they are done,
# so that neither has to be remembered.

CORE     := modules/libs/core
DESKTOP  := modules/apps/desktop
LANDING  := modules/apps/landing
ICON     := modules/tools/icon
UI       := modules/libs/ui
PROTOCOL := modules/libs/protocol

# What has to be on PATH, and who needs it:
#
#   go, node, npm                everything
#   pkg-config, gtk4,            the window, which links against the system's
#   webkitgtk-6.0                own browser through cgo
#   buf, protoc-gen-go,          the schema, and only for somebody changing it:
#   protoc-gen-connect-go        what they produce is committed
#
# How they get there is the machine's business. Nothing here provides them, so
# nothing here is slower than the work it does.

.PHONY: help
help:
	@grep -hE '^[a-z-]+:.*##' $(MAKEFILE_LIST) | sed 's/:.*##/\t/' | expand -t20

# Somebody adding a dependency writes the lock; a machine building from one
# does not: `make install INSTALL="npm ci"`.
INSTALL ?= npm install

.PHONY: install
install: ## fetch every module's dependencies
	cd $(PROTOCOL) && $(INSTALL)
	cd $(UI) && $(INSTALL)
	cd $(DESKTOP)/ui && $(INSTALL)
	cd $(DESKTOP)/flashcards && $(INSTALL)
	cd $(LANDING) && $(INSTALL)

.PHONY: generate
generate: ## compile the schema into Go and TypeScript
	cd $(PROTOCOL) && buf generate

.PHONY: generate-check
generate-check: generate ## fail if what is committed is out of date
	git add -AN -- $(PROTOCOL)
	git diff --exit-code -- $(PROTOCOL)

.PHONY: build
build: interface ## build everything
	cd $(CORE) && go build ./...
	cd $(DESKTOP) && go build ./...

.PHONY: interface
interface: ## build each window's page into the binary's assets
	cd $(UI) && npm run build
	cd $(DESKTOP)/ui && npm run build
	cd $(DESKTOP)/flashcards && npm run build

# On macOS the window is built into a bundle, ad-hoc signed. A bare executable
# there starts with no Dock tile and no menu bar.
ifeq ($(shell uname -s),Darwin)
# bundle <application> <binary> <plist> <icon>
define bundle
	rm -rf "$(1).app"
	mkdir -p "$(1).app/Contents/MacOS" "$(1).app/Contents/Resources"
	mv $(2) "$(1).app/Contents/MacOS/$(2)"
	cp $(DESKTOP)/build/darwin/$(3) "$(1).app/Contents/Info.plist"
	cp $(DESKTOP)/build/darwin/$(4) "$(1).app/Contents/Resources/$(4)"
	printf 'APPL????' > "$(1).app/Contents/PkgInfo"
	plutil -lint "$(1).app/Contents/Info.plist"
	codesign --force -s - "$(1).app"
endef
else
define bundle
endef
endif

.PHONY: desktop
desktop: interface ## build the window
	cd $(DESKTOP) && CGO_ENABLED=1 go build -o ../../../numen ./cmd/numen
	$(call bundle,Numen,numen,Info.plist,numen.icns)

.PHONY: flashcards
flashcards: interface ## build the window a person runs their cards in
	cd $(DESKTOP) && CGO_ENABLED=1 go build -o ../../../numen-flashcards ./cmd/numen-flashcards
	$(call bundle,Numen Flashcards,numen-flashcards,numen-flashcards-Info.plist,numen-flashcards.icns)

.PHONY: landing
landing: ## build the page the product is read about on
	cd $(LANDING) && npm run build

.PHONY: shoot
shoot: ## take the landing page's picture of the window from its story
	cd $(LANDING) && npm run shoot

.PHONY: icons
icons: ## cut every platform's icon from the one drawing
	cd $(ICON) && npm install && npm run build

# The window's tests reach the library through its build, so the library is
# built before they run.
.PHONY: test
test: ## run every test
	cd $(CORE) && go test ./... -race
	cd $(DESKTOP) && go test ./... -race
	cd $(UI) && npm test
	cd $(UI) && npm run build
	cd $(DESKTOP)/ui && npm test
	cd $(DESKTOP)/flashcards && npm test

.PHONY: lint
lint: generate-check ## the checks CI runs, less the one needing a base branch
	cd $(CORE) && gofmt -l . && go vet ./...
	cd $(DESKTOP) && gofmt -l ./cmd ./internal && go vet ./...
	cd $(PROTOCOL) && buf lint
	cd $(UI) && npm run typecheck
	cd $(DESKTOP)/ui && npm run typecheck
	cd $(DESKTOP)/flashcards && npm run typecheck
	cd $(LANDING) && npm run typecheck
