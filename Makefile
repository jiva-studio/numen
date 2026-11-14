# Go, Node, golangci-lint, buf and the Go protobuf plugins are taken from PATH.
# `nix develop` puts the versions CI runs there (see flake.nix); `make tool_versions`
# prints the ones on PATH.

# ---------------------------------- Modules -----------------------------------
CORE     := libs/core/kernel
ATOMICFILE := libs/atomicfile
NOTE    := libs/note
PLUGIN_CATALOG := libs/plugin/catalog
UI       := libs/ui/components
WORKBENCH := libs/ui/workbench
CORE_CLIENT := libs/core/client
PLUGIN_LOADER := libs/plugin/loader
PLUGIN_MANAGER := libs/plugin/manager
PROTOCOL := libs/core/protocol
PLUGIN_SDK := libs/plugin/sdk-ts
PLUGIN_SDK_GO := libs/plugin/sdk-go
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
PLUGIN_INDEX := tools/build-plugin-index
PLUGIN_BUNDLE := tools/bundle-plugins
CHECKS   := tools/checks
QUALITY  := tools/lint-quality
PLUGINS  := plugins
# Every directory under plugins/ holding a manifest.json is a plugin.
PLUGIN_DIRS := $(patsubst %/manifest.json,%,$(wildcard $(PLUGINS)/*/manifest.json))
TESTS    := tests

INSTALL  ?= npm install

# The race detector runs in CI only: it makes the Go tests 3 to 4 times slower. CI passes GO_RACE=1.
GO_RACE  ?=
LAYERS_FLAGS ?=
# Selection of `make checks`: TAGS=a,b (and), TAGS=a|b (or), EXCLUDE=tag, ONLY=names, CHANGED=1, LIST=1, SELFTEST=1.
TAGS     ?=
EXCLUDE  ?=
ONLY     ?=
CHANGED  ?=
LIST     ?=
SELFTEST ?=
CHECKS_ARGS = $(if $(TAGS),--tags '$(TAGS)') $(if $(EXCLUDE),--exclude '$(EXCLUDE)') $(if $(ONLY),--only '$(ONLY)') $(if $(CHANGED),--changed) $(if $(LIST),--list) $(if $(SELFTEST),--selftest)
GO_TEST_FLAGS := $(if $(GO_RACE),-race,)

# How many jobs the parallel targets (the gate, the plugin lint) run at once.
JOBS     ?= $(shell nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)
# Each golangci-lint and vue-tsc of a plugin uses every core, so the plugin lint runs half as many at once.
PLUGIN_LINT_JOBS ?= $(shell expr \( $(JOBS) + 1 \) / 2)
# GNU make 4 keeps the output of a job together; make 3.81 on macOS has no such option.
OUTPUT_SYNC := $(if $(filter output-sync,$(.FEATURES)),--output-sync=target,)

EXE      ?=
ifeq ($(OS),Windows_NT)
EXE      := .exe
endif

# ------------------------------------ Help ------------------------------------
.PHONY: help
help: ## show available targets
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_-]+:.*?## / {printf "  \033[36m%-28s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# ------------------------------- Workflow & CI --------------------------------
.PHONY: tool_versions
tool_versions: ## print the versions of the tools on PATH (nix develop provides the pinned ones)
	go version
	node --version
	golangci-lint --version
	buf --version
	protoc-gen-go --version
	protoc-gen-connect-go --version

.PHONY: install
install: ## fetch all workspace dependencies
	$(INSTALL)

.PHONY: compile
compile: lib_core_compile lib_atomicfile_compile lib_note_compile lib_plugin_catalog_compile lib_protocol_compile lib_ui_components_compile lib_plugin_sdk_compile lib_core_client_compile lib_workbench_compile lib_plugin_loader_compile lib_plugin_manager_compile lib_plugin_sdk_go_compile app_desktop_compile app_mobile_compile app_docs_compile app_landing_compile ## compile all modules and typecheck interfaces

.PHONY: build
build: interface lib_core_build app_desktop_build ## build all application artifacts

.PHONY: interface
interface: lib_workbench_build app_desktop_editor_build app_desktop_flashcards_build ## build UI pages into assets

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

# The index of the plugins is written next to their folders before anything compiled or packaged
# from them reads it.
.PHONY: plugins_index
plugins_index: ## write plugins/plugins-index.json from the manifest of every plugin
	node $(PLUGIN_INDEX)/build.mjs

.PHONY: plugins_index_test
plugins_index_test: ## test the plugin index build
	node --test $(PLUGIN_INDEX)/build.test.mjs $(PLUGIN_BUNDLE)/select.test.mjs

.PHONY: plugins_build
plugins_build: lib_ui_components_build plugins_index ## build every plugin
	$(call run_in_plugins,build)

.PHONY: plugins_lint
plugins_lint: lib_ui_components_build ## lint every plugin, $(PLUGIN_LINT_JOBS) at a time
	$(MAKE) -j $(PLUGIN_LINT_JOBS) $(OUTPUT_SYNC) $(PLUGIN_LINT_TARGETS)

PLUGIN_LINT_TARGETS := $(addprefix plugins_lint/,$(PLUGIN_DIRS))
.PHONY: $(PLUGIN_LINT_TARGETS)
$(PLUGIN_LINT_TARGETS): plugins_lint/%:
	$(MAKE) -C $* lint

.PHONY: plugins_test
plugins_test: lib_ui_components_build ## test every plugin
	$(call run_in_plugins,test)

.PHONY: plugins_clean
plugins_clean: ## remove what every plugin built
	$(call run_in_plugins,clean)

.PHONY: plugins_stories
plugins_stories: lib_ui_components_build ## draw the stories of every plugin, one engine at a time
	node $(STORIES)/run.mjs $(PLUGIN_SDK)/stories plugins

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
lib_core_test: plugins_index ## run core tests (GO_RACE=1 adds the race detector, which CI turns on)
	cd $(CORE) && go test ./... $(GO_TEST_FLAGS)

.PHONY: lib_core_lint
lib_core_lint: ## run linters for core
	$(CORE)/adapter/index/migration-forward-only.sh --self-test
	cd $(CORE) && go vet ./...
	cd $(CORE) && golangci-lint run ./... && golangci-lint run -c $(CURDIR)/.golangci-structure.yml ./...

.PHONY: lib_core_vulncheck
lib_core_vulncheck: ## scan core dependencies for vulnerabilities
	cd $(CORE) && go run golang.org/x/vuln/cmd/govulncheck@latest ./...

# ----------------------------- Lib: Atomic File ------------------------------
.PHONY: lib_atomicfile_build
lib_atomicfile_build: ## build the Go library that replaces a file whole or not at all
	cd $(ATOMICFILE) && go build ./...

.PHONY: lib_atomicfile_compile
lib_atomicfile_compile: lib_atomicfile_build

.PHONY: lib_atomicfile_test
lib_atomicfile_test: ## run atomic file tests
	cd $(ATOMICFILE) && go test ./... $(GO_TEST_FLAGS)

.PHONY: lib_atomicfile_lint
lib_atomicfile_lint: ## run linters for atomic file
	cd $(ATOMICFILE) && go vet ./...
	cd $(ATOMICFILE) && golangci-lint run ./...

# -------------------------------- Lib: Notes ---------------------------------
.PHONY: lib_note_build
lib_note_build: ## build the Go library of what a note is and how it is written
	cd $(NOTE) && go build ./...

.PHONY: lib_note_compile
lib_note_compile: lib_note_build

.PHONY: lib_note_test
lib_note_test: ## run notes tests
	cd $(NOTE) && go test ./... $(GO_TEST_FLAGS)

.PHONY: lib_note_lint
lib_note_lint: ## run linters for notes
	cd $(NOTE) && go vet ./...
	cd $(NOTE) && golangci-lint run ./... && golangci-lint run -c $(CURDIR)/.golangci-structure.yml ./...

# ------------------------- Lib: Plugin Catalog --------------------------------
.PHONY: lib_plugin_catalog_build
lib_plugin_catalog_build: ## build the Go library of plugin manifests and the plugin catalog
	cd $(PLUGIN_CATALOG) && go build ./...

.PHONY: lib_plugin_catalog_compile
lib_plugin_catalog_compile: lib_plugin_catalog_build

.PHONY: lib_plugin_catalog_test
lib_plugin_catalog_test: ## run plugin catalog tests
	cd $(PLUGIN_CATALOG) && go test ./... $(GO_TEST_FLAGS)

.PHONY: lib_plugin_catalog_lint
lib_plugin_catalog_lint: ## run linters for the plugin catalog
	cd $(PLUGIN_CATALOG) && go vet ./...
	cd $(PLUGIN_CATALOG) && golangci-lint run ./... && golangci-lint run -c $(CURDIR)/.golangci-structure.yml ./...

# ------------------------------- Lib: Protocol --------------------------------
.PHONY: lib_protocol_generate protocol_gen
lib_protocol_generate: ## generate protocol code and the lists of slots, when variables, shared commands and events
	cd $(PROTOCOL) && buf generate
	node $(PROTOCOL)/tools/slots.mjs
	node $(PROTOCOL)/tools/when.mjs
	node $(PROTOCOL)/tools/commands.mjs
	node $(PROTOCOL)/tools/events.mjs

protocol_gen: lib_protocol_generate ## alias for lib_protocol_generate

.PHONY: lib_protocol_generate_check
lib_protocol_generate_check: lib_protocol_generate ## verify protocol schema is up to date
	git add -AN -- $(PROTOCOL)/gen $(PROTOCOL)/src libs/plugin/catalog/manifest/slots.go libs/plugin/sdk-ts/src/slots/slotNames.ts libs/plugin/catalog/when/variables.go libs/plugin/sdk-ts/src/slots/whenVariables.ts libs/plugin/catalog/manifest/shared_commands.go libs/plugin/sdk-go/command/shared_commands.go libs/plugin/sdk-ts/src/commands/sharedCommands.ts libs/plugin/catalog/manifest/events.go
	git diff --exit-code -- $(PROTOCOL)/gen $(PROTOCOL)/src libs/plugin/catalog/manifest/slots.go libs/plugin/sdk-ts/src/slots/slotNames.ts libs/plugin/catalog/when/variables.go libs/plugin/sdk-ts/src/slots/whenVariables.ts libs/plugin/catalog/manifest/shared_commands.go libs/plugin/sdk-go/command/shared_commands.go libs/plugin/sdk-ts/src/commands/sharedCommands.ts libs/plugin/catalog/manifest/events.go

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
.PHONY: lib_ui_components_build
lib_ui_components_build: ## build @numen/ui package (SKIP_UI_BUILD=1 leaves a build that an earlier step made)
	$(if $(SKIP_UI_BUILD),@echo "$(UI) is built",cd $(UI) && npm run build)

.PHONY: lib_ui_components_typecheck
lib_ui_components_typecheck: ## typecheck UI components
	cd $(UI) && npm run typecheck

.PHONY: lib_ui_components_compile
lib_ui_components_compile: lib_ui_components_typecheck lib_ui_components_build

.PHONY: lib_ui_components_test
lib_ui_components_test: lib_ui_components_build ## run UI unit tests
	cd $(UI) && npm run test:unit

.PHONY: lib_ui_components_stories
lib_ui_components_stories: ## draw the stories of the UI components, one engine at a time
	cd $(UI) && npm run test:stories

.PHONY: lib_ui_components_lint
lib_ui_components_lint: ## lint UI components
	cd $(UI) && npm run lint
	cd $(UI) && npm run typecheck

.PHONY: lib_ui_components_knip
lib_ui_components_knip: ## check dead code and unused dependencies with knip
	npx knip --workspace $(UI)

# --------------------------------- Lib: Shell ---------------------------------
.PHONY: lib_workbench_build
lib_workbench_build: lib_ui_components_build ## build @numen/workbench package
	cd $(WORKBENCH) && npm run build

.PHONY: lib_workbench_typecheck
lib_workbench_typecheck: lib_ui_components_build ## typecheck the shell package
	cd $(WORKBENCH) && npm run typecheck

.PHONY: lib_workbench_compile
lib_workbench_compile: lib_workbench_typecheck lib_workbench_build

.PHONY: lib_workbench_test
lib_workbench_test: lib_ui_components_build ## run shell unit tests
	cd $(WORKBENCH) && npm run test:unit

.PHONY: lib_workbench_stories
lib_workbench_stories: lib_ui_components_build ## draw the stories of the shell, one engine at a time
	cd $(WORKBENCH) && npm run test:stories

.PHONY: lib_workbench_lint
lib_workbench_lint: lib_ui_components_build ## lint the shell package
	cd $(WORKBENCH) && npm run lint && npm run typecheck

.PHONY: lib_workbench_knip
lib_workbench_knip: ## check dead code and unused dependencies of the shell with knip
	npx knip --workspace $(WORKBENCH)

# ------------------------------ Lib: Core Client ------------------------------
.PHONY: lib_core_client_typecheck
lib_core_client_typecheck: ## typecheck the core-client package
	cd $(CORE_CLIENT) && npm run typecheck

.PHONY: lib_core_client_compile
lib_core_client_compile: lib_core_client_typecheck

.PHONY: lib_core_client_test
lib_core_client_test: ## run core-client tests
	cd $(CORE_CLIENT) && npm test

.PHONY: lib_core_client_lint
lib_core_client_lint: ## lint the core-client package
	cd $(CORE_CLIENT) && npm run lint && npm run typecheck

.PHONY: lib_core_client_knip
lib_core_client_knip: ## check dead code and unused dependencies of the core client with knip
	npx knip --workspace $(CORE_CLIENT)

# ------------------------------ Lib: Plugin Host ------------------------------
.PHONY: lib_plugin_loader_typecheck
lib_plugin_loader_typecheck: ## typecheck the plugin-loader package
	cd $(PLUGIN_LOADER) && npm run typecheck

.PHONY: lib_plugin_loader_compile
lib_plugin_loader_compile: lib_plugin_loader_typecheck

.PHONY: lib_plugin_loader_test
lib_plugin_loader_test: ## run plugin-loader tests
	cd $(PLUGIN_LOADER) && npm test

.PHONY: lib_plugin_loader_lint
lib_plugin_loader_lint: ## lint the plugin-loader package
	cd $(PLUGIN_LOADER) && npm run lint && npm run typecheck

# ----------------------------- Lib: Plugin Manager ----------------------------
.PHONY: lib_plugin_manager_typecheck
lib_plugin_manager_typecheck: lib_ui_components_build ## typecheck the plugin-manager package
	cd $(PLUGIN_MANAGER) && npm run typecheck

.PHONY: lib_plugin_manager_compile
lib_plugin_manager_compile: lib_plugin_manager_typecheck

.PHONY: lib_plugin_manager_test
lib_plugin_manager_test: lib_ui_components_build ## run plugin-manager tests
	cd $(PLUGIN_MANAGER) && npm test

.PHONY: lib_plugin_manager_lint
lib_plugin_manager_lint: lib_ui_components_build ## lint the plugin-manager package
	cd $(PLUGIN_MANAGER) && npm run lint && npm run typecheck

.PHONY: lib_plugin_manager_knip
lib_plugin_manager_knip: ## check dead code and unused dependencies of the plugin manager with knip
	npx knip --workspace $(PLUGIN_MANAGER)

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
lib_plugin_sdk_go_build: ## build plugin/sdk-go Go library
	for m in $(PLUGIN_SDK_GO_MODULES); do (cd $$m && go build ./...) || exit 1; done

.PHONY: lib_plugin_sdk_go_compile
lib_plugin_sdk_go_compile: lib_plugin_sdk_go_build

.PHONY: lib_plugin_sdk_go_test
lib_plugin_sdk_go_test: ## run plugin/sdk-go tests
	for m in $(PLUGIN_SDK_GO_MODULES); do (cd $$m && go test ./... $(GO_TEST_FLAGS)) || exit 1; done

.PHONY: lib_plugin_sdk_go_lint
lib_plugin_sdk_go_lint: ## run linters for plugin/sdk-go
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
# A plugin folder goes into the application whole. These are what it holds only for development: the
# sources of the backend and the frontend, their tests and configuration, and the archive that
# plugins_package writes into dist.
PLUGIN_BUNDLE_EXCLUDES := $(foreach pattern,node_modules backend frontend testdata coverage '*.go' go.mod go.sum Makefile package.json package-lock.json tsconfig.json 'vite.config.*' 'vitest.config.*' 'eslint.config.*' '.golangci*.yml' README.md 'dist/*.zip',--exclude $(pattern))
define bundle
	rm -rf "$(1).app"
	mkdir -p "$(1).app/Contents/MacOS" "$(1).app/Contents/Resources/plugins"
	mv $(2) "$(1).app/Contents/MacOS/$(2)"
	cp $(DESKTOP)/build/darwin/$(3) "$(1).app/Contents/Info.plist"
	cp $(DESKTOP)/build/darwin/$(4) "$(1).app/Contents/Resources/$(4)"
	for name in $$(node $(PLUGIN_BUNDLE)/select.mjs --target $(5)); do \
		mkdir -p "$(1).app/Contents/Resources/plugins/$$name"; \
		tar -C "$(PLUGINS)/$$name" -cf - $(PLUGIN_BUNDLE_EXCLUDES) . | tar -C "$(1).app/Contents/Resources/plugins/$$name" -xf -; \
		for bin in "$(1).app/Contents/Resources/plugins/$$name/bin/"*; do \
			if [ -f "$$bin" ] && [ -x "$$bin" ]; then \
				codesign --force -s - "$$bin"; \
			fi; \
		done; \
	done
	cp $(PLUGINS)/plugins-index.json "$(1).app/Contents/Resources/plugins/"
	node $(PLUGIN_BUNDLE)/select.mjs --target $(5) --verify "$(1).app/Contents/Resources/plugins"
	printf 'APPL????' > "$(1).app/Contents/PkgInfo"
	plutil -lint "$(1).app/Contents/Info.plist"
	codesign --force -s - "$(1).app"
endef
else
define bundle
endef
endif

.PHONY: app_desktop_editor_build
app_desktop_editor_build: lib_workbench_build
	cd $(DESKTOP)/editor && npm run build

.PHONY: app_desktop_flashcards_build
app_desktop_flashcards_build: lib_workbench_build
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
app_desktop_compile: lib_workbench_build ## compile desktop Go code and typecheck frontends
	cd $(DESKTOP) && go build ./...
	cd $(DESKTOP)/editor && npm run typecheck
	cd $(DESKTOP)/flashcards && npm run typecheck
	cd $(TESTS)/e2e/desktop && npm run lint
	$(MAKE) app_desktop_acceptance_lint

.PHONY: app_desktop_test
app_desktop_test: lib_workbench_build ## run desktop tests
	cd $(DESKTOP) && go test ./... $(GO_TEST_FLAGS)
	cd $(DESKTOP)/editor && npm test
	cd $(DESKTOP)/flashcards && npm test

.PHONY: app_desktop_stories
app_desktop_stories: lib_workbench_build ## draw the stories of the editor, one engine at a time
	cd $(DESKTOP)/editor && npm run test:stories

.PHONY: app_desktop_e2e
app_desktop_e2e: lib_workbench_build app_desktop_editor_build app_desktop_flashcards_build plugins_build ## run desktop end-to-end tests
	cd $(DESKTOP) && go build -tags numen_debug -o numen-debug ./cmd/numen
	cd $(DESKTOP) && go build -tags numen_debug -o numen-flashcards-debug ./cmd/numen-flashcards
	cd $(TESTS)/e2e/desktop && npm test
	cd $(TESTS)/e2e/desktop && npm run test:flashcards

.PHONY: app_desktop_acceptance
app_desktop_acceptance: lib_workbench_build app_desktop_editor_build app_desktop_flashcards_build plugins_build ## run the acceptance scenarios
	cd $(DESKTOP) && go build -tags numen_debug -ldflags "-X 'github.com/jiva-studio/numen/apps/desktop/internal/version.version=1.0.0 (build 1000, commit acceptance)'" -o numen-debug ./cmd/numen
	cd $(DESKTOP) && go build -tags numen_debug -o numen-flashcards-debug ./cmd/numen-flashcards
	cd plugins/connections-llm && go build -o bin/fakeopenai ./backend/cmd/fakeopenai
	cd $(TESTS)/acceptance/desktop && npm run acceptance:test

.PHONY: app_desktop_acceptance_lint
app_desktop_acceptance_lint: ## lint the acceptance scenarios, steps and driver
	cd $(TESTS)/acceptance/desktop && npm run acceptance:lint

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
	cd $(MOBILE) && go test ./... $(GO_TEST_FLAGS)
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

.PHONY: checks
checks: ## run the registered checks of tools/checks (usage: make checks [TAGS=fast,ts|go] [EXCLUDE=slow] [ONLY=go/layers] [CHANGED=1] [LIST=1] [SELFTEST=1]; TAGS=? lists the tags)
	node $(CHECKS)/run.mjs $(CHECKS_ARGS)

.PHONY: checks_selftest
checks_selftest: ## run the planted-violation tests of the registered checks
	node $(CHECKS)/run.mjs --selftest $(CHECKS_ARGS)

.PHONY: lint_layers check_layers tool_layers_check
lint_layers check_layers tool_layers_check: ## run the structural checks (alias of make checks TAGS=structure; LAYERS_FLAGS=--without-lint --without-go-tests)
	node $(CHECKS)/run.mjs --tags structure $(LAYERS_FLAGS) $(CHECKS_ARGS)

.PHONY: check_dependencies
check_dependencies: ## run dependency-cruiser over the frontend sources: package boundaries, segment direction, import cycles
	node $(CHECKS)/run.mjs --only ts/dependency-graph

.PHONY: check_styles
check_styles: ## run stylelint over every stylesheet and component style: tokens, no colour literal, no var() fallback
	node $(CHECKS)/run.mjs --only ts/styles

# Dead code is judged across packages: the apps import the shell through its build, so it is built first.
.PHONY: check_quality
check_quality: lib_workbench_build ## validate code quality metrics (usage: make check_quality [check=<name>] [opts=...])
	node $(QUALITY)/check.mjs $(if $(check),--check=$(check),) $(opts)

.PHONY: check_knip
check_knip: lib_workbench_build ## check dead code and unused dependencies of every TS package and plugin with knip
	npx knip

.PHONY: lint_narrative
lint_narrative: ## reject narrative identifiers, test names, comments, documents and the strings a person reads across the repository
	node $(CHECKS)/run.mjs --selftest --only style/names
	node $(CHECKS)/run.mjs --only style/names

.PHONY: lint_comments
lint_comments: ## reject comments on fields and properties and doc comments on unexported functions (Go, TypeScript, Vue)
	node $(CHECKS)/run.mjs --selftest --only style/comments
	node $(CHECKS)/run.mjs --only style/comments


.PHONY: check_typecheck
check_typecheck: lib_ui_components_build ## typecheck every frontend package and plugin with vue-tsc
	$(MAKE) typecheck_packages

# What check_typecheck does once the UI library is built, so the gate can build the library once.
.PHONY: typecheck_packages
typecheck_packages: typecheck_workspaces typecheck_plugins

.PHONY: typecheck_workspaces
typecheck_workspaces:
	npm run typecheck --workspaces --if-present

.PHONY: typecheck_plugins
typecheck_plugins:
	printf '%s\n' plugins/*/ | xargs -P $(JOBS) -I{} sh -c 'if [ -f "{}package.json" ] && grep -q "\"typecheck\"" "{}package.json"; then cd "{}" && npm run typecheck; fi'

# The generated code and the UI library are made first, because every part reads them; the parts are
# independent and run $(JOBS) at a time.
.PHONY: check_gate
check_gate: lib_protocol_generate_check lib_ui_components_build ## run every architectural check and the quality gates on changed files (pre-push gate)
	$(MAKE) -j $(JOBS) $(OUTPUT_SYNC) gate_parts

.PHONY: gate_parts
gate_parts: typecheck_workspaces typecheck_plugins lint_narrative lint_comments gate_layers gate_quality

.PHONY: gate_layers
gate_layers:
	node $(CHECKS)/run.mjs --tags structure --without-lint --without-go-tests

.PHONY: gate_quality
gate_quality:
	node $(QUALITY)/check.mjs --changed-only $(opts)

.PHONY: gate
gate: check_gate ## alias for check_gate

.PHONY: check_all
check_all: lib_ui_components_build ## run full architectural and quality suites across all modules
	$(MAKE) -j $(JOBS) $(OUTPUT_SYNC) all_parts

.PHONY: all_parts
all_parts: gate_layers all_quality all_narrative lint_comments

.PHONY: all_quality
all_quality:
	node $(QUALITY)/check.mjs $(opts)

.PHONY: all_narrative
all_narrative:
	node $(CHECKS)/run.mjs --only style/names

