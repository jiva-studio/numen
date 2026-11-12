# Landing

The page the product is read about on. Astro, static output, no framework on
the page: what is served is HTML, one stylesheet and two pictures.

```
npm install
npm run dev        # http://localhost:4321
npm run build      # dist/
npm run typecheck  # astro check
```

## The demo window

The window the page opens when "Try interactive demo" is pressed is `src/components/demo/`. It is a picture of the Numen window: the panes and tabs of `@numen/workbench` over fixed example data in `lib/` (a note, a file tree, a plex, a deck, a paper, transcripts). Nothing in it reads or changes a vault, runs a command or keeps a layout of its own; only what the tabs show moves (the open folders, the page of the paper, the playback, the canned answer of the agent). `model/` holds that playback state and `ui/` the views. It reaches the libraries `@numen/ui` and `@numen/workbench` only. The views of the tabs are small components drawn for the page, not the plugins' own views, because an application may not import a plugin.

## The picture of the window

The screenshots in `src/assets/` are taken from the stories that draw each
screen — the window in `libs/ui/components`, and a plugin's tab in the plugins'
Storybook — so the page shows the components the application is built from,
in the state they settle in. `tools/shots.mjs` names them.

```
npm run shoot
```

That starts the Storybooks the shots are declared in, takes each in light and
in dark at two and a half times the pixels, writes them into `src/assets/` as
WebP, and stops the Storybooks again. `STORYBOOK_PORT`,
`STORYBOOK_SHELL_PORT` and `STORYBOOK_PLUGINS_PORT` name ones already running
instead. The shell's and the plugins' Storybooks reach `@numen/ui` through its
build, so `npm run build` in `libs/ui/components` comes first.

The browser it drives is whichever Chrome is already on the machine, else
Playwright's own — the same rule the story tests follow. `CHROME_PATH` names
one directly:

```
CHROME_PATH=/path/to/chrome npm run shoot
```

Change what is in the picture by changing the story, not by editing the file.

## Where it is published

`site` in `astro.config.mjs` is what canonical URLs are built from, and is the
one thing here that has to agree with the DNS record.
