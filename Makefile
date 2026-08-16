# Handles, not a build system.
#
# Each module builds itself with its own tools — Go with go, the interface with
# npm. What is here is the list of things worth doing and where they are done,
# so that neither has to be remembered.

DESKTOP  := modules/apps/desktop
UI       := modules/libs/ui
PROTOCOL := modules/libs/protocol

# The plugins the schema is compiled with. They are needed only by someone
# changing it: what they produce is committed.
GENERATE := nix-shell -p buf protoc-gen-go protoc-gen-connect-go --run

# The window links against the system's own browser, so it is built with a
# compiler that can see GTK.
WEBVIEW := nix-shell -p pkg-config gtk4 webkitgtk_6_0 --run

.PHONY: help
help:
	@grep -hE '^[a-z-]+:.*##' $(MAKEFILE_LIST) | sed 's/:.*##/\t/' | expand -t20

.PHONY: install
install: ## fetch every module's dependencies
	cd $(PROTOCOL) && npm install
	cd $(UI) && npm install
	cd $(DESKTOP)/ui && npm install

.PHONY: generate
generate: ## compile the schema into Go and TypeScript
	cd $(PROTOCOL) && $(GENERATE) 'buf generate'

.PHONY: generate-check
generate-check: generate ## fail if what is committed is out of date
	git add -AN -- $(PROTOCOL)
	git diff --exit-code -- $(PROTOCOL)

.PHONY: build
build: interface ## build everything
	cd $(DESKTOP) && go build ./...

.PHONY: interface
interface: ## build the window's page into the binary's assets
	cd $(UI) && npm run build
	cd $(DESKTOP)/ui && npm run build

.PHONY: desktop
desktop: interface ## build the window
	cd $(DESKTOP) && $(WEBVIEW) 'CGO_ENABLED=1 go build -o ../../../numen ./cmd/numen'

.PHONY: test
test: ## run every test
	cd $(DESKTOP) && go test ./... -race
	cd $(UI) && npm test
	cd $(DESKTOP)/ui && npm test

.PHONY: lint
lint: generate-check ## the checks CI runs, less the one needing a base branch
	cd $(DESKTOP) && gofmt -l ./cmd ./internal && go vet ./...
	cd $(PROTOCOL) && $(GENERATE) 'buf lint'
	cd $(UI) && npm run typecheck
	cd $(DESKTOP)/ui && npm run typecheck
