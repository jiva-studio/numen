# Landing

The page the product is read about on. Astro, static output, no framework on
the page: what is served is HTML, one stylesheet and two pictures.

```
npm install
npm run dev        # http://localhost:4321
npm run build      # dist/
npm run typecheck  # astro check
```

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

## The plugin catalog

`/plugins` and `/plugins/<id>` are built from the plugin registry
(`docs/plugins/registry.md`), loaded by the content collection of
`src/content.config.ts`. The build reads the file NUMEN_PLUGIN_REGISTRY_FILE
names, or else `https://dl.numen.md/plugins/registry/latest.json`, and fails
when it cannot read either. For a preview:

```
node ../../tools/plugin-registry/collect.mjs --out ../../.registry --offline
NUMEN_PLUGIN_REGISTRY_FILE=../../.registry/latest.json npm run dev
```

`tools/fixtures/plugin-registry.json` is a small registry for the same purpose.

The pages are HTML. The search and the category links of the catalog are a
small script (`src/scripts/plugin-catalog.ts`) that hides the rows they do not
pass. A plugin icon is the lucide icon the registry names, drawn as inline SVG
at build time from the icon data of `@lucide/vue`.

## Where it is published

`site` in `astro.config.mjs` is what canonical URLs are built from, and is the
one thing here that has to agree with the DNS record.
