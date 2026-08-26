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

## Four pages write themselves

| Page | Written from |
| --- | --- |
| `keyboard.md` | the window's table of chords, `ui/src/keying.ts` |
| `commands.md` | every row of `ui/src/commanding.ts`, in the words `words.ts` draws them with |
| `reference.md` | the settings structs: `settings`, `embed`, `recognition`, `proofreading`, `agent` |
| `starting.md` | the flags `cmd/numen/main.go` declares |

Each is one block between `<!-- BEGIN AUTOGEN -->` and `<!-- END AUTOGEN -->`;
the prose around it is written by hand.

```
npm run manual        # write the blocks again
npm run manual:check  # fail where a page has drifted
```

`npm run build` writes them first, and the workflow checks them, so a command,
a chord, a setting or a flag added to the application without the manual
catching up is caught in CI.
