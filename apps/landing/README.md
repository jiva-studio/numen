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

The screenshots in `src/assets/` are declared in the stories that draw each
screen, with `app: 'landing'`, so the page shows the components the
application is built from, in the state they settle in.

```
npm run shoot
```

That runs `tools/shoot/` for this page: it starts the Storybooks the shots
are declared in, takes each in light and in dark, writes them into
`src/assets/` as WebP, and stops the Storybooks again. Its README says how a
story declares a screenshot, which Storybooks must be built first, and how
`CHROME_PATH` and the `STORYBOOK_*_PORT` variables are used.

Change what is in the picture by changing the story, not by editing the file.

## Where it is published

`site` in `astro.config.mjs` is what canonical URLs are built from, and is the
one thing here that has to agree with the DNS record.
