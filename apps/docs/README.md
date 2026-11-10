# The manual

The manual for a person using the application: how to write, link, find and ask. Astro and Starlight, static output, published at [docs.numen.md](https://docs.numen.md).

```
npm install
npm run dev        # http://localhost:4321
npm run build      # dist/
npm run typecheck  # astro check
```

## What belongs here

Only what somebody using numen needs. The specifications under `docs/` in this repository — the note format, the index, the performance targets — are how the application is built, and stay there.

## Five pages are generated

| Page           | Generated from                                                                                                                                                                                                                                                                                                       |
| -------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `keyboard.md`  | the keybindings of the application in `libs/core/protocol/commands.json`, and the keybindings the plugins declare                                                                                                                                                                                   |
| `commands.mdx` | the window's own commands, built in `editor/src/features/command-palette/lib/commands/table.ts` and labelled with the words in `shared/ui/words.ts`; then what the plugins add: the settings and commands their `manifest.json` offers the palette, the menu of the open file, and the commands each frontend registers |
| `reference.md` | the settings the core declares, read from `container.Settings`                                                                                                                                                                                                                                                       |
| `starting.md`  | the flags `cmd/numen/main.go` declares                                                                                                                                                                                                                                                                               |
| `cli.md`       | what `numen-cli` prints for its help, in `adapter/cli/cli.go`                                                                                                                                                                                                                                                        |

What the plugins add is read by `tools/ci-commands/commands.mjs`, out of each plugin's `manifest.json` and `frontend/index.ts`: the application cannot load a plugin, so the manual takes the list from that tool.

Each page has one block between `<!-- BEGIN AUTOGEN -->` and `<!-- END AUTOGEN -->`; the prose around it is written by hand.

```
npm run manual        # write the blocks again
npm run manual:check  # fail where a page is out of date
```

`npm run build` writes them first, and the workflow checks them, so a command, a chord, a setting or a flag added to the application without a matching manual change fails in CI.

## Taking the screenshots

```
npm run shoot  # every screenshot in src/assets/, light and dark
```

Each screenshot is a story, rendered at its real size in the Storybook that declares it: the components' Storybook in `libs/ui/components`, the shell's in `libs/ui/workbench` (which also renders the windows' screens), or the plugins' in `plugins/`. The run starts each Storybook a shot names and stops it afterwards. `STORYBOOK_PORT`, `STORYBOOK_SHELL_PORT` and `STORYBOOK_PLUGINS_PORT` name Storybooks that are already running. The shell's and the plugins' Storybooks use `@numen/ui` through its build, so run `npm run build` in `libs/ui/components` first. A screen of the desktop imports `@numen/protocol`, so that module needs its dependencies installed as well. A story that fails to render stops the run.

A screenshot whose pixels are unchanged is not rewritten: the encoder writes different bytes for the same image, so the run compares pixels. A run over an unchanged manual leaves the folder untouched.
