# The manual

What a person using the application reads: how to write, link, find and ask.
Astro and Starlight, static output, published at
[docs.numen.md](https://docs.numen.md).

```
npm install
npm run dev        # http://localhost:4321
npm run build      # dist/
npm run typecheck  # astro check
```

## What belongs here

Only what somebody using numen needs. The specifications under `docs/` in this
repository — the note format, the index, the performance targets, the decision
records — are how the application is built, and stay there.

## The keyboard page writes itself

The table of chords on `keyboard.md` is generated from the window's own table,
`modules/apps/desktop/ui/src/keying.ts`, and the words the commands are drawn
with. A key printed here is a key the application answers.

```
npm run keys        # write the block again
npm run keys:check  # fail where the page has drifted
```

`npm run build` writes it first, and the workflow checks it, so a chord added
to the window without the page catching up is caught in CI.
