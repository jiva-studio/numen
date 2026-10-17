# ---------------------------------- Modules -----------------------------------
MODULES  := modules
CORE     := modules/libs/core
UI       := modules/libs/ui
WIRE     := modules/libs/wire
PROTOCOL := modules/libs/protocol
PLUGIN_SDK := modules/libs/plugin-sdk
PLUGIN_SDK_GO := modules/libs/plugin-sdk-go

DESKTOP  := modules/apps/desktop
DESKTOP_MONOLITH := modules/apps/desktop-monolith
MOBILE   := modules/apps/mobile
DOCS     := modules/apps/docs
LANDING  := modules/apps/landing

ICON     := modules/tools/icon
STORIES  := modules/tools/stories
PLUGINS  := modules/plugins
PLUGIN_NAMES := $(notdir $(patsubst %/,%,$(dir $(wildcard $(PLUGINS)/*/manifest.json))))

INSTALL  ?= npm install

EXE      ?=
ifeq ($(OS),Windows_NT)
EXE      := .exe
endif

# ------------------------------------ Help ------------------------------------
.PHONY: help
help: ## show available targets
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_-]+:.*?## / {printf "  \033[36m%-28s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# ------------------------------- Workflow & CI --------------------------------
.PHONY: install
install: ## fetch all workspace dependencies
	cd $(MODULES) && $(INSTALL)

.PHONY: compile
compile: lib_core_compile lib_protocol_compile lib_ui_compile lib_wire_compile lib_plugin_sdk_go_compile app_desktop_compile app_mobile_compile app_docs_compile app_landing_compile ## compile all modules and typecheck interfaces

.PHONY: build
build: interface lib_core_build app_desktop_build ## build all application artifacts

.PHONY: interface
interface: lib_ui_build app_desktop_editor_build app_desktop_flashcards_build ## build UI pages into assets

.PHONY: generate
generate: lib_protocol_generate ## generate code from schema

.PHONY: generate-check
generate-check: lib_protocol_generate_check ## verify generated code matches schema

.PHONY: desktop
desktop: app_desktop_build ## build desktop application

.PHONY: desktop_monolith
desktop_monolith: interface ## build desktop application in monolithic mode
	cd $(DESKTOP_MONOLITH) && CGO_ENABLED=1 go build -tags plugins_monolith -o ../../../numen .
	$(call bundle,Numen,numen,Info.plist,numen.icns)

.PHONY: desktop_modular
desktop_modular: interface plugins_build ## build desktop application in modular mode
	cd $(DESKTOP) && CGO_ENABLED=1 go build $(TAGS_FLAG) -o ../../../numen ./cmd/numen
	$(call bundle,Numen,numen,Info.plist,numen.icns)

.PHONY: plugins
plugins: plugins_build ## build all native plugins

.PHONY: plugins_build
plugins_build: $(addprefix plugin_,$(addsuffix _build,$(PLUGIN_NAMES))) ## build all native plugins

.PHONY: plugins_lint
plugins_lint: $(addprefix plugin_,$(addsuffix _lint,$(PLUGIN_NAMES))) ## lint all native plugins

.PHONY: plugins_test
plugins_test: $(addprefix plugin_,$(addsuffix _test,$(PLUGIN_NAMES))) ## run tests across all plugins

.PHONY: plugins_package
plugins_package: $(addprefix plugin_,$(addsuffix _package,$(PLUGIN_NAMES))) ## package all plugins into zip archives

.PHONY: flashcards
flashcards: app_desktop_flashcards_bin ## build flashcards standalone window

.PHONY: landing
landing: app_landing_build ## build landing site

.PHONY: shoot
shoot: app_landing_shoot ## take landing screenshots

.PHONY: icons
icons: tool_icon_build ## cut application icons

# --------------------------------- Lib: Core ----------------------------------
.PHONY: lib_core_build
lib_core_build: ## build core Go library
	cd $(CORE) && go build ./...

.PHONY: lib_core_compile
lib_core_compile: lib_core_build

.PHONY: lib_core_test
lib_core_test: ## run core tests with race detector
	cd $(CORE) && go test ./... -race

.PHONY: lib_core_lint
lib_core_lint: ## run linters for core
	$(CORE)/adapter/index/migration-forward-only.sh --self-test
	cd $(CORE) && go vet ./...
	cd $(CORE) && golangci-lint run ./...

.PHONY: lib_core_vulncheck
lib_core_vulncheck: ## scan core dependencies for vulnerabilities
	cd $(CORE) && go run golang.org/x/vuln/cmd/govulncheck@latest ./...

# ------------------------------- Lib: Protocol --------------------------------
.PHONY: lib_protocol_generate
lib_protocol_generate: ## generate protocol code
	cd $(PROTOCOL) && buf generate

.PHONY: lib_protocol_generate_check
lib_protocol_generate_check: lib_protocol_generate ## verify protocol schema is up to date
	git add -AN -- $(PROTOCOL)/gen $(PROTOCOL)/src
	git diff --exit-code -- $(PROTOCOL)/gen $(PROTOCOL)/src

.PHONY: lib_protocol_compile
lib_protocol_compile: ## compile protocol Go packages
	cd $(PROTOCOL) && go build ./...

.PHONY: lib_protocol_test
lib_protocol_test: lib_protocol_compile

.PHONY: lib_protocol_lint
lib_protocol_lint: ## lint protocol definitions
	cd $(PROTOCOL) && buf lint
	cd $(PROTOCOL) && golangci-lint run ./...

.PHONY: lib_protocol_vulncheck
lib_protocol_vulncheck: ## scan protocol dependencies for vulnerabilities
	cd $(PROTOCOL) && go run golang.org/x/vuln/cmd/govulncheck@latest ./...

# ---------------------------------- Lib: UI -----------------------------------
.PHONY: lib_ui_build
lib_ui_build: ## build @numen/ui package
	cd $(UI) && npm run build

.PHONY: lib_ui_typecheck
lib_ui_typecheck: ## typecheck UI components
	cd $(UI) && npm run typecheck

.PHONY: lib_ui_compile
lib_ui_compile: lib_ui_typecheck lib_ui_build

.PHONY: lib_ui_test
lib_ui_test: lib_ui_build ## run UI unit tests
	cd $(UI) && npm run test:unit

.PHONY: lib_ui_lint
lib_ui_lint: ## lint UI components
	cd $(UI) && npm run lint
	cd $(UI) && npm run typecheck

.PHONY: lib_ui_knip
lib_ui_knip: ## check dead code and unused dependencies with knip
	cd $(UI) && npx --yes knip

# --------------------------------- Lib: Wire ----------------------------------
.PHONY: lib_wire_typecheck
lib_wire_typecheck: ## typecheck wire package
	cd $(WIRE) && npm run typecheck

.PHONY: lib_wire_compile
lib_wire_compile: lib_wire_typecheck

.PHONY: lib_wire_test
lib_wire_test: ## run wire tests
	cd $(WIRE) && npm test

.PHONY: lib_wire_lint
lib_wire_lint: ## lint wire package
	cd $(WIRE) && npm run typecheck

# ------------------------------ Lib: Plugin SDK -------------------------------
.PHONY: lib_plugin_sdk_typecheck
lib_plugin_sdk_typecheck: ## typecheck plugin-sdk package
	cd $(PLUGIN_SDK) && npm run typecheck

.PHONY: lib_plugin_sdk_compile
lib_plugin_sdk_compile: lib_plugin_sdk_typecheck

.PHONY: lib_plugin_sdk_test
lib_plugin_sdk_test: ## run plugin-sdk unit tests
	cd $(PLUGIN_SDK) && npm test

.PHONY: lib_plugin_sdk_lint
lib_plugin_sdk_lint: lib_plugin_sdk_typecheck
 
# ---------------------------- Lib: Plugin SDK Go ------------------------------
.PHONY: lib_plugin_sdk_go_build
lib_plugin_sdk_go_build: ## build plugin-sdk-go Go library
	cd $(PLUGIN_SDK_GO) && go build ./...

.PHONY: lib_plugin_sdk_go_compile
lib_plugin_sdk_go_compile: lib_plugin_sdk_go_build

.PHONY: lib_plugin_sdk_go_test
lib_plugin_sdk_go_test: ## run plugin-sdk-go tests with race detector
	cd $(PLUGIN_SDK_GO) && go test ./... -race

.PHONY: lib_plugin_sdk_go_lint
lib_plugin_sdk_go_lint: ## run linters for plugin-sdk-go
	cd $(PLUGIN_SDK_GO) && go vet ./...
	cd $(PLUGIN_SDK_GO) && golangci-lint run ./...

# ------------------------------ Plugins Tooling -------------------------------
.PHONY: create_plugin
create_plugin: ## scaffold a new plugin (usage: make create_plugin name=<id> [template=<reader|compute|agent>] [type=<native|frontend>])
	cd $(DESKTOP) && go run ./cmd/numen-plugin create $(name) --template=$(template) --type=$(type)

# -------------------------------- App: Desktop --------------------------------
BUILD_TAGS ?=
TAGS_FLAG :=
ifneq ($(BUILD_TAGS),)
TAGS_FLAG := -tags $(BUILD_TAGS)
endif

ifeq ($(shell uname -s),Darwin)
define bundle
	rm -rf "$(1).app"
	mkdir -p "$(1).app/Contents/MacOS" "$(1).app/Contents/Resources"
	mv $(2) "$(1).app/Contents/MacOS/$(2)"
	cp $(DESKTOP)/build/darwin/$(3) "$(1).app/Contents/Info.plist"
	cp $(DESKTOP)/build/darwin/$(4) "$(1).app/Contents/Resources/$(4)"
	mkdir -p "$(1).app/Contents/PlugIns"
	for p in $$(find $(PLUGINS) -mindepth 1 -maxdepth 1 -type d); do \
		if [ -f "$$p/manifest.json" ] && [ -d "$$p/bin" ]; then \
			name=$$(basename "$$p"); \
			mkdir -p "$(1).app/Contents/PlugIns/$$name"; \
			cp "$$p/manifest.json" "$(1).app/Contents/PlugIns/$$name/"; \
			cp -r "$$p/bin" "$(1).app/Contents/PlugIns/$$name/"; \
			if [ -d "$$p/dist" ]; then cp -r "$$p/dist" "$(1).app/Contents/PlugIns/$$name/"; fi; \
			for bin in "$(1).app/Contents/PlugIns/$$name/bin/"*; do \
				if [ -f "$$bin" ] && [ -x "$$bin" ]; then \
					codesign --force -s - "$$bin"; \
				fi; \
			done; \
		fi; \
	done
	printf 'APPL????' > "$(1).app/Contents/PkgInfo"
	plutil -lint "$(1).app/Contents/Info.plist"
	codesign --force -s - "$(1).app"
endef
else
define bundle
endef
endif

.PHONY: app_desktop_editor_build
app_desktop_editor_build: lib_ui_build
	cd $(DESKTOP)/editor && npm run build

.PHONY: app_desktop_flashcards_build
app_desktop_flashcards_build: lib_ui_build
	cd $(DESKTOP)/flashcards && npm run build

.PHONY: app_desktop_build
app_desktop_build: interface ## build desktop binary
	cd $(DESKTOP) && CGO_ENABLED=1 go build $(TAGS_FLAG) -o ../../../numen ./cmd/numen
	$(call bundle,Numen,numen,Info.plist,numen.icns)

.PHONY: app_desktop_pgo
app_desktop_pgo: ## refresh the desktop PGO profile from the benchmarks
	$(DESKTOP)/build/pgo.sh

.PHONY: app_desktop_pgo_check
app_desktop_pgo_check: ## fail when the desktop PGO profile names a local path
	$(DESKTOP)/build/pgo.sh --check

.PHONY: app_desktop_flashcards_bin
app_desktop_flashcards_bin: interface ## build desktop flashcards binary
	cd $(DESKTOP) && CGO_ENABLED=1 go build -o ../../../numen-flashcards ./cmd/numen-flashcards
	$(call bundle,Numen Flashcards,numen-flashcards,numen-flashcards-Info.plist,numen-flashcards.icns)

.PHONY: app_desktop_compile
app_desktop_compile: lib_ui_build ## compile desktop Go code and typecheck frontends
	cd $(DESKTOP) && go build ./...
	cd $(DESKTOP)/editor && npm run typecheck
	cd $(DESKTOP)/flashcards && npm run typecheck

.PHONY: app_desktop_test
app_desktop_test: lib_ui_build ## run desktop tests
	cd $(DESKTOP) && go test ./... -race
	cd $(DESKTOP)/editor && npm test
	cd $(DESKTOP)/flashcards && npm test

.PHONY: app_desktop_e2e
app_desktop_e2e: lib_ui_build app_desktop_editor_build plugin_reader-epub_build plugin_reader-pdf_build plugin_transcription_build ## run desktop end-to-end tests
	cd $(DESKTOP) && go build -tags debug -o numen-debug ./cmd/numen
	cd $(MODULES)/tests/e2e/desktop && npx playwright test

.PHONY: app_desktop_lint
app_desktop_lint: ## lint desktop module
	cd $(DESKTOP) && go vet ./...
	cd $(DESKTOP) && golangci-lint run ./...
	cd $(DESKTOP)/editor && npm run lint
	cd $(DESKTOP)/editor && npm run typecheck
	cd $(DESKTOP)/flashcards && npm run lint
	cd $(DESKTOP)/flashcards && npm run typecheck

.PHONY: app_desktop_vulncheck
app_desktop_vulncheck: ## scan desktop dependencies for vulnerabilities
	cd $(DESKTOP) && go run golang.org/x/vuln/cmd/govulncheck@latest ./...

# -------------------------------- App: Mobile ---------------------------------
.PHONY: app_mobile_build
app_mobile_build: ## build mobile module
	cd $(MOBILE) && go build ./...

.PHONY: app_mobile_compile
app_mobile_compile: ## compile mobile module
	cd $(MOBILE) && go build ./...
	cd $(MOBILE) && npm run typecheck

.PHONY: app_mobile_test
app_mobile_test: ## run mobile tests
	cd $(MOBILE) && go test ./... -race
	cd $(MOBILE) && npm test

.PHONY: app_mobile_lint
app_mobile_lint: ## lint mobile module
	cd $(MOBILE) && go vet ./...
	cd $(MOBILE) && golangci-lint run ./...
	cd $(MOBILE) && npm run lint
	cd $(MOBILE) && npm run typecheck

.PHONY: app_mobile_vulncheck
app_mobile_vulncheck: ## scan mobile dependencies for vulnerabilities
	cd $(MOBILE) && go run golang.org/x/vuln/cmd/govulncheck@latest ./...

# --------------------------------- App: Docs ----------------------------------
.PHONY: app_docs_compile
app_docs_compile: ## typecheck docs module
	cd $(DOCS) && npm run typecheck

.PHONY: app_docs_lint
app_docs_lint: ## lint docs module
	cd $(DOCS) && npm run lint
	cd $(DOCS) && npm run manual:check
	cd $(DOCS) && npm run typecheck

# -------------------------------- App: Landing --------------------------------
.PHONY: app_landing_build
app_landing_build: ## build landing site
	cd $(LANDING) && npm run build

.PHONY: app_landing_shoot
app_landing_shoot: ## take landing screenshots
	cd $(LANDING) && npm run shoot

.PHONY: app_landing_compile
app_landing_compile: ## typecheck landing site
	cd $(LANDING) && npm run typecheck

.PHONY: app_landing_lint
app_landing_lint: ## lint landing site
	cd $(LANDING) && npm run lint
	cd $(LANDING) && npm run typecheck
	cd $(LANDING) && npm run stories:check

# -------------------------------- Tools & Meta --------------------------------
.PHONY: tool_icon_build
tool_icon_build: ## cut application icons
	cd $(ICON) && npm run build

.PHONY: tool_stories_check
tool_stories_check: ## verify story references
	cd $(STORIES) && npm run check

# ---------------------------------- Plugins -----------------------------------
.PHONY: $(addprefix plugin_,$(addsuffix _build,$(PLUGIN_NAMES)))
$(addprefix plugin_,$(addsuffix _build,$(PLUGIN_NAMES))): plugin_%_build: plugin_%_backend_build plugin_%_frontend_build

.PHONY: $(addprefix plugin_,$(addsuffix _backend_build,$(PLUGIN_NAMES)))
$(addprefix plugin_,$(addsuffix _backend_build,$(PLUGIN_NAMES))): plugin_%_backend_build:
	@if [ -d $(PLUGINS)/$*/backend ]; then \
		mkdir -p $(PLUGINS)/$*/bin && \
		cd $(PLUGINS)/$* && go build -o bin/$*$(EXE) ./backend ; \
	fi

.PHONY: $(addprefix plugin_,$(addsuffix _frontend_build,$(PLUGIN_NAMES)))
$(addprefix plugin_,$(addsuffix _frontend_build,$(PLUGIN_NAMES))): plugin_%_frontend_build:
	@if [ -f $(PLUGINS)/$*/package.json ]; then \
		cd $(PLUGINS)/$* && npm run build ; \
	elif [ -d $(PLUGINS)/$*/frontend ]; then \
		mkdir -p $(PLUGINS)/$*/dist ; \
		if [ -f $(PLUGINS)/$*/frontend/styles.css ]; then \
			cp $(PLUGINS)/$*/frontend/styles.css $(PLUGINS)/$*/dist/styles.css ; \
		fi ; \
	fi

.PHONY: $(addprefix plugin_,$(addsuffix _lint,$(PLUGIN_NAMES)))
$(addprefix plugin_,$(addsuffix _lint,$(PLUGIN_NAMES))): plugin_%_lint:
	$(MAKE) -C $(PLUGINS)/$* lint

.PHONY: $(addprefix plugin_,$(addsuffix _test,$(PLUGIN_NAMES)))
$(addprefix plugin_,$(addsuffix _test,$(PLUGIN_NAMES))): plugin_%_test:
	$(MAKE) -C $(PLUGINS)/$* test

.PHONY: $(addprefix plugin_,$(addsuffix _package,$(PLUGIN_NAMES)))
$(addprefix plugin_,$(addsuffix _package,$(PLUGIN_NAMES))): plugin_%_package: plugin_%_build
	cd $(DESKTOP) && go run ./cmd/numen-plugin package ../../../$(PLUGINS)/$*

.PHONY: $(addprefix plugin_,$(addsuffix _clean,$(PLUGIN_NAMES)))
$(addprefix plugin_,$(addsuffix _clean,$(PLUGIN_NAMES))): plugin_%_clean:
	rm -rf $(PLUGINS)/$*/bin $(PLUGINS)/$*/dist

.PHONY: plugins_clean
plugins_clean: $(addprefix plugin_,$(addsuffix _clean,$(PLUGIN_NAMES))) ## clean all plugin artifacts
