# @numen/ui

Interface components shared by the desktop and mobile clients.

**The rules are in [`coding-style-frontend.md`](../../../.agents/rules/coding-style-frontend.md).** What follows is where they are applied in this tree.

## The one rule

**A component here uses nothing from the application and says nothing of its domain.** No import from the application's domain, no wire types, nothing from `apps/**` — not in the components, not in the stories, not in the fixtures — and no domain word in a name, a prop, an event or a story: a component says `tree`, `menu`, `row`, `slot`, `item`, never `vault` or `note`. A component takes props it defines itself and emits events with opaque identifiers; whoever renders it maps its domain onto these general words and maps the identifier back. The window (or `@numen/workbench`) that draws a list of vaults passes it as `items` of a `tree` or a `menu`.

The dependency runs one way: `apps/*` depends on `libs/ui/components`. Never the reverse, and never sideways. `@numen/ui` is a private package (`"private": true`, on no registry) whose public contract plugins build on, so its names are general ones. It holds no code of the desktop shell or of a domain, so its names stay general: the Wails runtime, and what loads or calls it, belongs to the window that is served by Wails.

## Working on it

```bash
npm install
npx playwright install chromium webkit
npm run storybook     # http://localhost:6006 — every component, every state
npm test
npm run typecheck
```

Storybook is the development environment. A component is built and reviewed there, in isolation, before any application renders it.

## The engines the stories run in

**A story is rendered in WebKit and in Chromium, because the window is both.** The desktop window is the platform's own webview: WebKit on a mac and on Linux, where it links against `webkitgtk-6.0`, and Chromium on Windows. A property that one engine supports and the other does not — `user-select` against `-webkit-user-select` — then fails a test here.

Chromium falls back to a browser already on the machine, which `CHROME_PATH` names. WebKit has no such fallback: Playwright uses only its own build, so install it with `npx playwright install webkit`.

**What the story level still does not prove.** Playwright's builds of the two engines are not the window's webview. The version is Playwright's and not the machine's, and none of what the window adds around the page is present.

On a distribution whose shared libraries Playwright's WebKit does not find — NixOS among them, where the build requires `libicudata.so.74` and the other libraries under Ubuntu's names — the story level cannot run on the machine at all. It runs in the image Playwright publishes for the version in `package.json`:

```bash
docker run --rm --ipc=host -v "$PWD:/ui" -w /ui \
  mcr.microsoft.com/playwright:v1.62.1-noble npx vitest run --project stories
```

## Layout

```
src/
  index.ts           the public entry: everything a consumer imports
  shared/
    tokens/          all styling
      tokens.css     every value, as custom properties
      theme.css      Tailwind names defined from those values
      classes.css    shared class lists
      window.css     the window frame
    lib/             pure helpers: async, dom, format, math, text, time (the clock)
    ui/              one folder per component: button, menu, select, slider, switch, ...
    fixtures/        made-up text and a clock for stories
  features/
    tree/            the tree component and its pure model
```

A component is a folder in `shared/ui/`; a part that has state of its own and is drawn on its own is a slice of `features/`. Placement is the skill [`nm-place-library`](../../../.agents/skills/nm-place-library/SKILL.md).

## Styling

**Two ways of writing a style, split by the kind of component.** A component made of DOM is styled with Tailwind utilities; a component that paints SVG is styled with scoped CSS, because the properties it paints with — `rx`, `stroke-dasharray`, `paint-order`, `r` — are SVG attributes that no utility expresses.

Both use the same tokens: `tokens.css` holds every value, `theme.css` defines Tailwind's names in terms of those tokens, and `bg-surface` and `var(--numen-surface)` are one value.

**A theme is chosen by `color-scheme`.** The tokens are `light-dark()` pairs, and that is the whole mechanism — Storybook's switch sets `color-scheme` and everything follows. It is declared on `:root` and **nowhere else**: an element that declares it again returns the choice to the system setting, so every token under that element resolves to the wrong half of its pair while the page appears to use the other one. **No component writes a `dark:` class.** Tailwind's `dark:` variant reads a class or the OS setting, neither of which is what is being switched, so a component using one stays in the theme it was written for. Components copied from shadcn-vue have theirs removed when they are copied in.

**Scoped CSS overrides a utility.** Component styles are unlayered and Tailwind's are in `@layer`, so an unlayered rule wins whatever its specificity. A scoped block on a DOM component is therefore for what a utility cannot express — an input field and its copy sharing one grid cell — and never for what one can.

**A component root has the class `numen`.** The tokens are declared on `:root` and on `.numen` both, so a component works in a page that never set them.
