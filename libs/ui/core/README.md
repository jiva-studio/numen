# @numen/ui

Interface components shared by the desktop and mobile clients.

**The rules are in [`coding-style-frontend.md`](../../../.agents/rules/coding-style-frontend.md).** What follows is where they live in this tree.

## The one rule

**A component here reaches nothing of the application.** No import from the application's domain, no wire types, nothing from `apps/**` — not in the components, not in the stories, not in the fixtures. A component takes props it defines itself and emits events carrying opaque identifiers; whoever renders it translates what it holds into that shape.

**A concept takes the word its field gives it, and that includes the domain's own.** Where a component draws a vault it says vault: the welcome page names its rows `vaults`, and `letters.ts` gives each of them the letter it is reached by. A shape word coined to keep the domain out of the module would be a second name for the thing on the screen, and the naming rule refuses one. What is refused is the dependency, not the vocabulary.

The dependency runs one way: `apps/*` depends on `libs/ui/core`. Never the reverse, and never sideways.

## Working on it

```bash
npm install
npx playwright install chromium webkit
npm run storybook     # http://localhost:6006 — every component, every state
npm test
npm run typecheck
```

Storybook is the development environment. A component is built and judged there, in isolation, before any application renders it.

## The engines the stories run in

**A story is rendered in WebKit and in Chromium, because the window is both.** The desktop window is the platform's own webview: WebKit on a mac and on Linux, where it links against `webkitgtk-6.0`, and Chromium on Windows. A property one engine reads and the other does not — `user-select` against `-webkit-user-select` — then fails a test here.

Chromium falls back to a browser already on the machine, which `CHROME_PATH` names. WebKit has no such fallback: Playwright drives its own build and nothing else, so `npx playwright install webkit` is what puts it there.

**What the story level still does not prove.** Playwright's builds of the two engines are not the window's webview. The version is Playwright's and not the machine's, and none of what the window adds around the page is there.

On a distribution whose shared libraries Playwright's WebKit does not find — NixOS among them, where the build asks for `libicudata.so.74` and the rest by Ubuntu's names — the story level cannot run on the machine at all. It runs in the image Playwright publishes for the version in `package.json`:

```bash
docker run --rm --ipc=host -v "$PWD:/ui" -w /ui \
  mcr.microsoft.com/playwright:v1.62.1-noble npx vitest run --project stories
```

## Layout

```
src/
  tokens/            the whole styling contract
    tokens.css       every value, as custom properties
    theme.css        Tailwind, told what those values are
  classes.ts         cn(), for joining class lists
  lib/               the clock, the geometry and the viewport, as parameters
  components/ui/     what shadcn-vue supplies, and this module now owns
  fixtures/          awkward text, and a backdrop for a panel to float over
  panel/             the floating panel: a surface to put things on
  composer/          the field a message is written in
    model.ts         what state it is in, and what a key means
  dots/              three dots rising in turn: something is being written
  thread/            the conversation
    model.ts         every voice, declared once — a new voice starts here
  assembled/         the three of them in one piece, for Storybook
```

## Styling

**Two ways of writing a style, along one line.** A component made of DOM is styled with Tailwind utilities; a component that paints SVG is styled with scoped CSS, because what it paints with — `rx`, `stroke-dasharray`, `paint-order`, `r` — are SVG attributes that no utility expresses.

Both end at the same place: `tokens.css` holds every value, `theme.css` defines Tailwind's names in terms of those tokens, and `bg-surface` and `var(--numen-surface)` are one value.

**A theme is chosen by `color-scheme`.** The tokens are `light-dark()` pairs, and that is the whole mechanism — Storybook's switch sets `color-scheme` and everything follows. It is declared on `:root` and **nowhere else**: an element that declares it again hands the choice back to the reader's system, and every token under that element resolves to the wrong half of its pair while the page looks like it asked for the other one. **No component writes a `dark:` class.** Tailwind's `dark:` variant reads a class or the OS setting, neither of which is what is being switched, so a component using one is stuck in whichever theme it was written in. Components taken from shadcn-vue have theirs removed on the way in.

**Scoped CSS beats a utility.** Component styles are unlayered and Tailwind's are in `@layer`, so an unlayered rule wins whatever its specificity. A scoped block on a DOM component is therefore for what a utility cannot say — the composer's field and its copy sharing one grid cell — and never for what one can.

**A component root carries `numen`.** The tokens are declared on `:root` and on `.numen` both, so a component works in a page that never set them.
