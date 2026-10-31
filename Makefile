# ---------------------------------- Modules -----------------------------------
CORE     := libs/core
NOTES    := libs/notes
PLUGIN_CATALOG := libs/plugin-catalog
UI       := libs/ui/core
SHELL_LIB := libs/ui/shell
PLUGIN_HOST := libs/plugin-host-ts
PROTOCOL := libs/protocol
PLUGIN_SDK := libs/plugin-sdk-ts
PLUGIN_SDK_GO := libs/plugin-sdk-go
# The base module of the Go plugin SDK and each feature module under it.
PLUGIN_SDK_GO_MODULES := $(patsubst %/go.mod,%,$(wildcard $(PLUGIN_SDK_GO)/go.mod $(PLUGIN_SDK_GO)/*/go.mod))

DESKTOP  := apps/desktop
MOBILE   := apps/mobile
DOCS     := apps/docs
LANDING  := apps/landing

ICON     := tools/build-icons
STORIES  := tools/ci-stories
COMMANDS := tools/ci-commands
PLUGIN_VERSIONS := tools/plugin-versions
LAYERS   := tools/lint-layers
QUALITY  := tools/lint-quality
NARRATIVE := tools/lint-narrative
PLUGINS  := plugins
# Every directory under plugins/ holding a manifest.json is a plugin.
PLUGIN_DIRS := $(patsubst %/manifest.json,%,$(wildcard $(PLUGINS)/*/manifest.json))
TESTS    := tests

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
	$(INSTALL)

.PHONY: compile
compile: lib_core_compile lib_notes_compile lib_plugin_catalog_compile lib_protocol_compile lib_ui_compile lib_plugin_sdk_compile lib_shell_compile lib_plugin_host_compile lib_plugin_sdk_go_compile app_desktop_compile app_mobile_compile app_docs_compile app_landing_compile ## compile all modules and typecheck interfaces

.PHONY: build
build: interface lib_core_build app_desktop_build ## build all application artifacts

.PHONY: interface
interface: lib_shell_build app_desktop_editor_build app_desktop_flashcards_build ## build UI pages into assets

.PHONY: generate
generate: lib_protocol_generate ## generate code from schema

.PHONY: generate-check
generate-check: lib_protocol_generate_check ## verify generated code matches schema

.PHONY: desktop
desktop: app_desktop_build ## build desktop application

.PHONY: desktop_modular
desktop_modular: interface plugins_build ## build desktop application in modular mode
	cd $(DESKTOP) && CGO_ENABLED=1 go build $(TAGS_FLAG) -o $(CURDIR)/numen ./cmd/numen
	$(call bundle,Numen,numen,Info.plist,numen.icns,desktop.editor)

# Each runs the target of that name in every plugin's own Makefile.
define run_in_plugins
	@set -e; for dir in $(PLUGIN_DIRS); do echo "$$dir"; $(MAKE) -C "$$dir" $(1); done
endef

.PHONY: plugins
plugins: plugins_build ## build every plugin

.PHONY: plugins_build
plugins_build: lib_ui_build ## build every plugin
	$(call run_in_plugins,build)

.PHONY: plugins_lint
plugins_lint: lib_ui_build ## lint every plugin
	$(call run_in_plugins,lint)

.PHONY: plugins_test
plugins_test: lib_ui_build ## test every plugin
	$(call run_in_plugins,test)

.PHONY: plugins_clean
plugins_clean: ## remove what every plugin built
	$(call run_in_plugins,clean)

.PHONY: plugins_stories
plugins_stories: lib_ui_build ## draw the stories of every plugin in Chromium and WebKit
	cd $(PLUGIN_SDK)/stories && npx vitest run

.PHONY: plugins_package
plugins_package: plugins_build ## package every plugin into a zip archive
	@set -e; for dir in $(PLUGIN_DIRS); do \
		(cd $(DESKTOP) && go run ./cmd/numen-plugin package $(CURDIR)/$$dir); \
	done

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
	cd $(CORE) && golangci-lint run ./... && golangci-lint run -c $(CURDIR)/.golangci-structure.yml ./...

.PHONY: lib_core_vulncheck
lib_core_vulncheck: ## scan core dependencies for vulnerabilities
	cd $(CORE) && go run golang.org/x/vuln/cmd/govulncheck@latest ./...

# -------------------------------- Lib: Notes ---------------------------------
.PHONY: lib_notes_build
lib_notes_build: ## build the Go library of what a note is and how it is written
	cd $(NOTES) && go build ./...

.PHONY: lib_notes_compile
lib_notes_compile: lib_notes_build

.PHONY: lib_notes_test
lib_notes_test: ## run notes tests with race detector
	cd $(NOTES) && go test ./... -race

.PHONY: lib_notes_lint
lib_notes_lint: ## run linters for notes
	cd $(NOTES) && go vet ./...
	cd $(NOTES) && golangci-lint run ./... && golangci-lint run -c $(CURDIR)/.golangci-structure.yml ./...

# ------------------------- Lib: Plugin Catalog --------------------------------
.PHONY: lib_plugin_catalog_build
lib_plugin_catalog_build: ## build the Go library of plugin manifests and the plugin catalog
	cd $(PLUGIN_CATALOG) && go build ./...

.PHONY: lib_plugin_catalog_compile
lib_plugin_catalog_compile: lib_plugin_catalog_build

.PHONY: lib_plugin_catalog_test
lib_plugin_catalog_test: ## run plugin catalog tests with race detector
	cd $(PLUGIN_CATALOG) && go test ./... -race

.PHONY: lib_plugin_catalog_lint
lib_plugin_catalog_lint: ## run linters for the plugin catalog
	cd $(PLUGIN_CATALOG) && go vet ./...
	cd $(PLUGIN_CATALOG) && golangci-lint run ./...

# ------------------------------- Lib: Protocol --------------------------------
.PHONY: lib_protocol_generate protocol_gen
lib_protocol_generate: ## generate protocol code
	cd $(PROTOCOL) && buf generate

protocol_gen: lib_protocol_generate ## alias for lib_protocol_generate

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
	cd $(PROTOCOL) && golangci-lint run ./... && golangci-lint run -c $(CURDIR)/.golangci-structure.yml ./...

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
	cd $(UI) && npm run knip

# --------------------------------- Lib: Shell ---------------------------------
.PHONY: lib_shell_build
lib_shell_build: lib_ui_build ## build @numen/shell package
	cd $(SHELL_LIB) && npm run build

.PHONY: lib_shell_typecheck
lib_shell_typecheck: lib_ui_build ## typecheck the shell package
	cd $(SHELL_LIB) && npm run typecheck

.PHONY: lib_shell_compile
lib_shell_compile: lib_shell_typecheck lib_shell_build

.PHONY: lib_shell_test
lib_shell_test: lib_ui_build ## run shell unit tests
	cd $(SHELL_LIB) && npm run test:unit

.PHONY: lib_shell_stories
lib_shell_stories: lib_ui_build ## draw the stories of the shell, one engine at a time
	@set -e; for engine in chromium webkit; do \
		(cd $(SHELL_LIB) && npx vitest run --project "stories ($$engine)"); \
	done

.PHONY: lib_shell_lint
lib_shell_lint: lib_ui_build ## lint the shell package
	cd $(SHELL_LIB) && npm run lint && npm run typecheck

.PHONY: lib_shell_knip
lib_shell_knip: ## check dead code and unused dependencies of the shell with knip
	cd $(SHELL_LIB) && npm run knip

# ------------------------------ Lib: Plugin Host ------------------------------
.PHONY: lib_plugin_host_typecheck
lib_plugin_host_typecheck: ## typecheck the plugin-host package
	cd $(PLUGIN_HOST) && npm run typecheck

.PHONY: lib_plugin_host_compile
lib_plugin_host_compile: lib_plugin_host_typecheck

.PHONY: lib_plugin_host_test
lib_plugin_host_test: ## run plugin-host tests
	cd $(PLUGIN_HOST) && npm test

.PHONY: lib_plugin_host_lint
lib_plugin_host_lint: ## lint the plugin-host package
	cd $(PLUGIN_HOST) && npm run lint && npm run typecheck

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
	for m in $(PLUGIN_SDK_GO_MODULES); do (cd $$m && go build ./...) || exit 1; done

.PHONY: lib_plugin_sdk_go_compile
lib_plugin_sdk_go_compile: lib_plugin_sdk_go_build

.PHONY: lib_plugin_sdk_go_test
lib_plugin_sdk_go_test: ## run plugin-sdk-go tests with race detector
	for m in $(PLUGIN_SDK_GO_MODULES); do (cd $$m && go test ./... -race) || exit 1; done

.PHONY: lib_plugin_sdk_go_lint
lib_plugin_sdk_go_lint: ## run linters for plugin-sdk-go
	for m in $(PLUGIN_SDK_GO_MODULES); do (cd $$m && go vet ./... && golangci-lint run ./... && golangci-lint run -c $(CURDIR)/$(PLUGIN_SDK_GO)/.golangci-structure.yml ./...) || exit 1; done

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
	mkdir -p "$(1).app/Contents/MacOS" "$(1).app/Contents/Resources/plugins"
	mv $(2) "$(1).app/Contents/MacOS/$(2)"
	cp $(DESKTOP)/build/darwin/$(3) "$(1).app/Contents/Info.plist"
	cp $(DESKTOP)/build/darwin/$(4) "$(1).app/Contents/Resources/$(4)"
	for p in $$(find $(PLUGINS) -mindepth 1 -maxdepth 1 -type d); do \
		if [ -f "$$p/manifest.json" ] && { [ -d "$$p/bin" ] || [ -d "$$p/dist" ]; } && grep -q '"$(5)"' "$$p/manifest.json"; then \
			name=$$(basename "$$p"); \
			mkdir -p "$(1).app/Contents/Resources/plugins/$$name"; \
			cp "$$p/manifest.json" "$(1).app/Contents/Resources/plugins/$$name/"; \
			if [ -d "$$p/bin" ]; then cp -r "$$p/bin" "$(1).app/Contents/Resources/plugins/$$name/"; fi; \
			if [ -d "$$p/dist" ]; then cp -r "$$p/dist" "$(1).app/Contents/Resources/plugins/$$name/"; fi; \
			for bin in "$(1).app/Contents/Resources/plugins/$$name/bin/"*; do \
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
app_desktop_editor_build: lib_shell_build
	cd $(DESKTOP)/editor && npm run build

.PHONY: app_desktop_flashcards_build
app_desktop_flashcards_build: lib_shell_build
	cd $(DESKTOP)/flashcards && npm run build

.PHONY: app_desktop_build
app_desktop_build: interface plugins_build ## build desktop binary
	cd $(DESKTOP) && CGO_ENABLED=1 go build $(TAGS_FLAG) -o $(CURDIR)/numen ./cmd/numen
	$(call bundle,Numen,numen,Info.plist,numen.icns,desktop.editor)

.PHONY: app_desktop_pgo
app_desktop_pgo: ## refresh the desktop PGO profile from the benchmarks
	$(DESKTOP)/build/pgo.sh

.PHONY: app_desktop_pgo_check
app_desktop_pgo_check: ## fail when the desktop PGO profile names a local path
	$(DESKTOP)/build/pgo.sh --check

.PHONY: app_desktop_flashcards_bin
app_desktop_flashcards_bin: app_desktop_flashcards_build plugins_build ## build desktop flashcards binary
	cd $(DESKTOP) && CGO_ENABLED=1 go build -o $(CURDIR)/numen-flashcards ./cmd/numen-flashcards
	$(call bundle,Numen Flashcards,numen-flashcards,numen-flashcards-Info.plist,numen-flashcards.icns,desktop.flashcards)

.PHONY: app_desktop_compile
app_desktop_compile: lib_shell_build ## compile desktop Go code and typecheck frontends
	cd $(DESKTOP) && go build ./...
	cd $(DESKTOP)/editor && npm run typecheck
	cd $(DESKTOP)/flashcards && npm run typecheck

.PHONY: app_desktop_test
app_desktop_test: lib_shell_build ## run desktop tests
	cd $(DESKTOP) && go test ./... -race
	cd $(DESKTOP)/editor && npm test
	cd $(DESKTOP)/flashcards && npm test

.PHONY: app_desktop_stories
app_desktop_stories: lib_shell_build ## draw the stories of the editor, one engine at a time
	@set -e; for engine in chromium webkit; do \
		(cd $(DESKTOP)/editor && npx vitest run --project "stories ($$engine)"); \
	done

.PHONY: app_desktop_e2e
app_desktop_e2e: lib_shell_build app_desktop_editor_build app_desktop_flashcards_build plugins_build ## run desktop end-to-end tests
	cd $(DESKTOP) && go build -tags numen_debug -o numen-debug ./cmd/numen
	cd $(DESKTOP) && go build -tags numen_debug -o numen-flashcards-debug ./cmd/numen-flashcards
	cd $(TESTS)/e2e/desktop && npm test
	cd $(TESTS)/e2e/desktop && npm run test:flashcards

.PHONY: app_desktop_lint
app_desktop_lint: ## lint desktop module
	cd $(DESKTOP) && go vet ./...
	cd $(DESKTOP) && golangci-lint run ./... && golangci-lint run -c $(CURDIR)/.golangci-structure.yml ./...
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
	cd $(MOBILE) && golangci-lint run ./... && golangci-lint run -c $(CURDIR)/.golangci-structure.yml ./...
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

.PHONY: tool_commands_check
tool_commands_check: ## verify how plugin commands are read for the manual
	cd $(COMMANDS) && npm run check

.PHONY: plugin_versions_check
plugin_versions_check: ## verify every changed plugin raised its <year>.<month>.<build> version (usage: make plugin_versions_check [base=origin/main])
	node $(PLUGIN_VERSIONS)/check.mjs --base $(or $(base),origin/main)

.PHONY: plugin_versions_bump
plugin_versions_bump: ## write the next version into every changed plugin (usage: make plugin_versions_bump [base=origin/main])
	node $(PLUGIN_VERSIONS)/check.mjs --base $(or $(base),origin/main) --write

.PHONY: plugin_versions_test
plugin_versions_test: ## test the plugin version check
	node --test $(PLUGIN_VERSIONS)/check.test.mjs

.PHONY: check_layers tool_layers_check
check_layers tool_layers_check: ## validate architectural layer boundaries (usage: make check_layers [target=libs|plugins|apps|<module>])
	node $(LAYERS)/check.mjs $(target)

.PHONY: check_quality
check_quality: ## validate code quality metrics (usage: make check_quality [check=<name>] [opts=...])
	node $(QUALITY)/check.mjs $(if $(check),--check=$(check),) $(opts)

.PHONY: lint_narrative
lint_narrative: ## reject narrative identifiers, test names and comments across the repository
	node $(NARRATIVE)/scan.mjs . --check

.PHONY: check_typecheck
check_typecheck: lib_ui_build ## typecheck every frontend package and plugin with vue-tsc
	npm run typecheck --workspaces --if-present
	for dir in plugins/*/; do if [ -f "$$dir/package.json" ] && grep -q '"typecheck"' "$$dir/package.json"; then (cd "$$dir" && npm run typecheck) || exit 1; fi; done

.PHONY: check_gate
check_gate: lib_protocol_generate_check check_typecheck ## run architectural and quality gates on changed files (fast pre-push gate)
	node $(LAYERS)/check.mjs --changed-only
	node $(QUALITY)/check.mjs --changed-only $(opts)

.PHONY: gate
gate: check_gate ## alias for check_gate

.PHONY: check_all
check_all: ## run full architectural and quality suites across all modules
	node $(LAYERS)/check.mjs
	node $(QUALITY)/check.mjs $(opts)
	node $(NARRATIVE)/scan.mjs . --check

