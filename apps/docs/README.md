# The manual

What a person using the application reads: how to write, link, find and ask. Astro and Starlight, static output, published at [docs.numen.md](https://docs.numen.md).

```
npm install
npm run dev        # http://localhost:4321
npm run build      # dist/
npm run typecheck  # astro check
```

## What belongs here

Only what somebody using numen needs. The specifications under `docs/` in this repository — the note format, the index, the performance targets — are how the application is built, and stay there.

## Five pages write themselves

| Page | Written from |
| --- | --- |
| `keyboard.md` | the window's table of chords, `editor/src/features/command-palette/lib/chords/chords.ts`, and the shortcuts the plugins register |
| `commands.mdx` | the window's own commands, composed in `editor/src/features/command-palette/lib/commands/table.ts` and drawn in the words of `shared/words.ts`; then what the plugins add — the settings and commands their `manifest.json` offers the palette, the menu of the file in front, and the commands each frontend registers |
| `reference.md` | the settings the core declares, walked from `container.Settings` |
| `starting.md` | the flags `cmd/numen/main.go` declares |
| `cli.md` | what `numen-cli` prints when it is asked, in `adapter/cli/cli.go` |

Each is one block between `<!-- BEGIN AUTOGEN -->` and `<!-- END AUTOGEN -->`; the prose around it is written by hand.

```
npm run manual        # write the blocks again
npm run manual:check  # fail where a page has drifted
```

`npm run build` writes them first, and the workflow checks them, so a command, a chord, a setting or a flag added to the application without the manual catching up is caught in CI.

## Taking the pictures

```
npm run shoot  # every picture in src/assets/, light and dark
```

Each is a story, drawn at the size it is drawn at, in the Storybook that declares it: the components' own in `libs/ui`, which draws the windows' screens too, or the plugins' in `plugins/`. Both are started for the run and stopped again; `STORYBOOK_PORT` and `STORYBOOK_PLUGINS_PORT` name ones already running. The plugins' Storybook reaches `@numen/ui` through its build, so `npm run build` in `libs/ui` comes first. A screen of the desktop reaches `@numen/protocol`, so that module needs its dependencies installed as well, and a story that will not draw stops the run rather than being photographed.

A picture that came back the same picture is left where it is: the encoder writes different bytes of one window, so what is compared is the pixels. A run over a manual nothing has changed under leaves the folder untouched.
