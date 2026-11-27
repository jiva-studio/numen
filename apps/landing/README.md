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

## Where it is published

`site` in `astro.config.mjs` is what canonical URLs are built from, and is the
one thing here that has to agree with the DNS record.
