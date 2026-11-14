# The plugin adapter and the applications

## The plugin adapter

`libs/core/kernel/internal/adapter/plugin` and its packages connect the core to plugin processes. They are two kinds of adapter, told apart by the package name.

- A package starting with `host` handles the calls a plugin makes to the core (`hostendpoint`, `hosttext`, `hostvault`, `hostsearch`). It is a driving adapter: it checks the plugin's permission, maps the call to the arguments of a use case, calls the use case and maps the result back. It references no type under `port/`.
- A package ending in `bridge` calls a plugin for the core (`commandbridge`, `providerbridge`, `sourcebridge`, `toolbridge`). It is a driven adapter: it implements a port a use case declared, sends the call over the transport and maps the plugin's answer to the port's types.
- `bridge`, `transport` and `manifestfile` hold what both kinds share: the wire to a plugin process and the reading of a manifest. They hold no rule about notes.
- The root package `plugin` is the host: roles (`Registry`, `Discovery`, `Activation`, `Retry`, `Teardown`, `Settings`, `ConfigPush`, `Listeners`, `ContentTypes`, `Events`, `Mode`, and from `supervision` the `Processes`, `Failures` and `Services`) that each keep their own state and are made by the container (`newPluginRoles`) with one constructor each, `NewRegistry`, `NewActivation` and so on, that takes a `Deps` struct of the roles made before it, so that none holds a pointer set after construction and none calls a role made after it. The package `plugin/supervision` keeps the supervisors, endpoints and clients of the plugin processes, starts and stops the processes. A role has its own small API, and `container` hands each use case the role it needs. What a change of a setting causes (stop the plugin, sync its listeners, send its configuration) is the use case `ApplyPluginSetting`, not a method of the host. A struct of the root package has at most the methods and fields `ThePluginAdapterHasNoGodStruct` allows.
- The core knows no plugin by name. A plugin is discovered through its manifest, and what it contributes (`tabKinds`, `settingsViews`, `slots`, `themes`, `readers`) is read from there.
- A plugin depends on the SDK only. The core never imports a plugin and a plugin never imports the core.
- A host call that only exists for one plugin is a call the SDK offers to every plugin, or it is not added. A mechanism lands with its first consumer.

A new host call: add the method on the matching `host*` package, add or reuse the use case it calls, add the SDK function and the first plugin that uses it in the same change. A new call from the core to a plugin: the use case declares the port, a `*bridge` package implements it, and `container` binds it.

## The applications

`apps/desktop` holds the Go host of the windows: `cmd/<binary>/main.go` parses flags and calls `app.Run`, `app` starts the toolkit, creates the windows and wires the toolkit to the container, and `internal/` holds pieces only this application uses.

- An application builds a `container.Config`, asks `container` for the created use cases and adapters, and routes requests to them. It holds no scenario and no rule about notes.
- A driven adapter an application owns (the folder dialog of the toolkit, the keychain) is its own package with one role. It implements a port and does not call a use case.
- A window's Go side serves the generated services by mapping requests to use cases; the mapping lives in the driving adapter of `libs/core/kernel/adapter/window`, and the application mounts it.
- Which plugins a window loads is decided by the target the binary names (`desktop.editor`, `desktop.flashcards`) and by the manifests; the application names no plugin.
- The flashcards binary differs from the editor by its target and its window; it shares the container and every use case.
