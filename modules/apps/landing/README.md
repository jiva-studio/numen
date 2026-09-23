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

The screenshot in `src/assets/` is taken from the story that draws the whole
window — `Application/Window` in `modules/libs/ui` — so the page shows the
components the application is built from, in the state they settle in.

```
npm run shoot
```

That starts Storybook, takes the window in light and in dark at twice the
pixels, writes both into `src/assets/` as WebP, and stops Storybook again.
`STORYBOOK_PORT` names one already running instead.

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
