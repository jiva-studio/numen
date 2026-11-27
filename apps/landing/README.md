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
