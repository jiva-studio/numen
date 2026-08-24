# ADR-0041: A theme is a CSS file, and there is one of them

- **Status:** Proposed
- **Date:** 2026-08-24
- **Applies to:** `modules/libs/ui`, `modules/apps/desktop`,
  `modules/libs/protocol`
- **Partly supersedes:** ADR-0013 — the settings file the application never
  writes
- **Related:** ADR-0000, ADR-0013, ADR-0017, ADR-0020, ADR-0024, ADR-0025,
  ADR-0028, ADR-0035

## Context

ADR-0028 settled that the tokens are `light-dark()` pairs and that setting
`color-scheme` changes every colour in the module at once. It left the person
out of it: the pair a token resolves to is decided by the operating system, and
the values in the pair are decided at build time.

Two things are wanted. The person picks light or dark themselves. The person
picks a palette that is not this product's, and can write one.

## Decision

### A theme is one CSS file that redeclares tokens under `:root`

```css
:root {
  color-scheme: dark;
  --numen-surface: #282a36;
  --numen-node-fg: #f8f8f2;
}
```

A file that names one token is a valid theme, and every token it leaves out
keeps what `tokens.css` set.

It is an ordinary stylesheet, applied whole. Nothing parses it, bounds it to
declarations, or holds it to a contrast floor. A file that hides the window is
the person's file.

### The tokens are declared on the root alone

`tokens.css` declares every token under `:where(:root), :where(.numen)` today,
and `numen` is the class every component carries. A custom property declared on
an element beats the same property inherited from the root, whatever the
selectors weigh, so a theme written against `:root` reaches `body` and nothing
inside a component.

The second half of that selector list goes. Components inherit from the root,
which is what makes a `:root` block a theme.

### The preset themes and the person's themes are the same thing

A preset ships inside the application. A person's theme is a `.css` file in
`numen/themes/` under the folder this desktop keeps its configuration in. Same
format, same list, same one line that applies either.

A preset copied out, edited and dropped in that folder stands in the list beside
the one it came from. That is what makes a theme writable.

### A theme is named by where it came from and what its file is called

`preset:dracula` ships here; `mine:dracula` is the person's file. Two themes can
carry one filename and neither hides the other.

A name resolves to a file inside the themes folder and nowhere else: `mine:` is
joined to that folder and refused if it leaves it.

### The folder is read flat, and only `.css` is a theme

One level, names ending `.css`. A folder inside it is not a theme and neither is
`dracula.css.bak`. The folder is made, empty, on the first run.

On a filesystem that does not tell case apart, `Dracula.css` and `dracula.css`
are one file. Nothing here works around that.

### Exactly one theme is applied

`appearance.theme` names it. Its text goes into a single `<style>` element,
replacing whatever was there. Nothing is layered, so there is no order to learn.

A name matching nothing applies `preset:numen` and says which name it could not
find, in the way ADR-0035 says such a thing is said. The file is not rewritten:
putting the theme back is all it takes.

### The chosen theme is written into `numen.json`

ADR-0013 says the application never writes that file, and ADR-0000 counts the
selected theme as application state. Both are set aside here: a person looking
for a setting looks in the file the settings are in, and a theme is a setting.

The file is patched as an object — read, one field set, written back — and never
marshalled from `settings.Config`, whose `embed.ServiceModel.MarshalJSON` and
`proofreading.Service.MarshalJSON` drop keys. A patch leaves every field the
application does not know exactly where it was.

The write is atomic and `0600`, which is what the file is today. A file that
does not parse is not patched.

### The theme declares whether it is a pair

A palette published in two halves writes `light-dark()` pairs and says nothing
about `color-scheme`; `appearance.mode` — `system`, `light`, `dark` — then
chooses. A palette published dark only writes flat colours and pins
`color-scheme: dark`.

While a theme that pins is applied, the light-and-dark control has nothing to
choose and is drawn as having nothing to choose.

### Both style elements are the last thing in the head

`app.css` declares `:root { color-scheme: light dark }` at ordinary weight and
lands last in the built bundle, whose `<link>` is the last element of the head.
A theme's `:root` weighs the same, so order alone decides.

The mode's element and the theme's go after that `<link>`, mode first. Replacing
a theme re-appends the theme's element only.

### The window opens already wearing it

The page is served with both elements already in it, so no frame shows the
default colours. `index.html` stops being served by the file server and gets a
handler that splices them in.

### The window asks the core for themes

The catalogue, a theme's text, the choice, and the folder having changed are
four things the window asks of the core, so they are four additions to the
schema, generated and committed as ADR-0025 requires.

### The editor's colours are the product's tokens

The nine syntax colours in `editor/theme.ts` are hardcoded pairs declared on the
CodeMirror element, where no theme reaches them. They become `--numen-*` tokens,
so a theme paints marked-up text in its own palette.

### A hover is a mix, not a brightness

`--numen-hover-brightness` is a number carried in `light-dark()`, which takes
colours only, so the filter it feeds is invalid and hover does nothing today.
The two places that read it take `color-mix()` against the token beneath, and
the token goes.

### Twelve presets ship, beside this product's own

`github`, `one-dark`, `dracula`, `monokai`, `ayu`, `tokyo-night`, `cobalt2`,
`nord`, `solarized`, `catppuccin` and `gruvbox` are their publishers' palettes,
taken from those publishers' own sources, syntax colours among them. What a
palette does not name is derived from what it does, and the file says which.

A seat is moved off a published colour where that colour is the one the focused
node is painted in. Those two stand side by side in the plex, and a palette
published for a text editor has no seats to keep apart.

`amber` is four colours read off a screenshot with the rest derived from them,
and its name is provisional until someone gives it one.

`preset:numen` names no token. `tokens.css` is this palette, and a theme naming
nothing keeps every value that file holds.

### The palette offers the theme, and shows it as it is chosen

A theme is one thing and the half it is read as is another, so they are two
commands in the band over the window. Each opens a step of its own — the themes
in one, `system`, `light` and `dark` in the other — and each applies what the
keyboard lands on. Leaving without choosing puts back what was there.

The themes stand in two bands, the ones that ship and the person's own. A row
says what is true of that row alone, and the shelf it came off is what the band
above it says.

`Palette.vue` emits what was chosen, gone back from and dismissed. It gains a
fourth, for where the keyboard is standing, and the step applies a theme from
it.

## Consequences

**Positive**

- A theme is a file a person can read, and a broken one is a file to delete.
- The light and dark setting costs one line: the tokens are already pairs and
  `color-scheme` is already what reads them.

**Negative**

- The tokens become a compatibility surface. Renaming one breaks every theme
  written against it.
- The application now writes a file a person types in.
- A theme can hide the window, and the way back is to edit the settings by hand.
- Contrast is the palette publisher's, so a preset can be less legible than what
  `tokens.css` holds.

## Alternatives considered

**A JSON theme against a published schema, the way VS Code and Zed do it.**
Rejected: every token would need a name in the schema and a mapping to the
custom property, and the two would drift.

**Obsidian's layering: a theme, then snippets over it.** Rejected for now. The
whole of it is a rule about which of several files wins.

**A theme per vault.** Rejected. The window would change colour because of which
folder was opened.

**Keeping only custom-property declarations and dropping the rest of a theme.**
Rejected. It would bound what a theme can break, and it would also stop a theme
doing anything the tokens do not already name.
