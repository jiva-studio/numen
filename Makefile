# Handles, not a build system.
#
# Each module builds itself with its own tools — Go with go, the interface with
# npm. What is here is the list of things worth doing and where they are done,
# so that neither has to be remembered.

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

.PHONY: install
install: ## fetch every module's dependencies
	cd $(PROTOCOL) && npm install
	cd $(UI) && npm install
	cd $(DESKTOP)/ui && npm install
	cd $(LANDING) && npm install

.PHONY: generate
generate: ## compile the schema into Go and TypeScript
	cd $(PROTOCOL) && buf generate

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
	cd $(DESKTOP) && CGO_ENABLED=1 go build -o ../../../numen ./cmd/numen

.PHONY: landing
landing: ## build the page the product is read about on
	cd $(LANDING) && npm run build

.PHONY: shoot
shoot: ## take the landing page's picture of the window from its story
	cd $(LANDING) && npm run shoot

.PHONY: icons
icons: ## cut every platform's icon from the one drawing
	cd $(ICON) && npm install && npm run build

.PHONY: test
test: ## run every test
	cd $(DESKTOP) && go test ./... -race
	cd $(UI) && npm test
	cd $(DESKTOP)/ui && npm test

.PHONY: lint
lint: generate-check ## the checks CI runs, less the one needing a base branch
	cd $(DESKTOP) && gofmt -l ./cmd ./internal && go vet ./...
	cd $(PROTOCOL) && buf lint
	cd $(UI) && npm run typecheck
	cd $(DESKTOP)/ui && npm run typecheck
	cd $(LANDING) && npm run typecheck
