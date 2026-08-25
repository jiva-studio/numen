# Themes

A theme is one CSS file that redeclares the tokens the interface draws with.
Some ship inside the application and the rest are files in a folder of the
person's; they are one list, one format, and one line applies either.

Every token a theme leaves out keeps the value `tokens.css` holds. A file
naming one token is a theme.

## The file

```css
:root {
  --numen-surface: #101014;
  --numen-node-bg: #17171d;
  --numen-node-fg: #e6e6ea;
}
```

Every token stands on the root, and a component inherits what it paints with,
so a `:root` block reaches every component. A custom property declared on an
element beats the same property inherited from the root, whatever the selectors
weigh, so `:root` is where a theme is written.

It is an ordinary stylesheet, applied whole. Nothing reads inside it beyond
whether it sets `color-scheme`. A file that hides the window is the person's
file, and the way back is `appearance.theme` in the settings.

## Light and dark

A palette published in two halves writes `light-dark()` pairs and says nothing
about `color-scheme`:

```css
:root {
  --numen-surface: light-dark(#fdf6e3, #002b36);
}
```

`appearance.mode` — `system`, `light`, `dark` — chooses which half is read.

A palette published in one half writes flat colours and pins the half it is:

```css
:root {
  color-scheme: dark;
  --numen-surface: #21222c;
}
```

Such a theme is listed as pinned, and the mode has nothing left to choose.
What counts as pinning is the property with a colon after it, anywhere in the
file; the comments are cut away first, so a comment saying `color-scheme` pins
nothing.

## Where a theme lives

The themes folder is `numen/themes/` in the folder this desktop keeps a
person's configuration in, beside `numen.json` — see [Settings](settings.md).
It is made, empty, on the first run.

| | |
| --- | --- |
| One level | a folder inside the themes folder is not a theme, and neither is a file one level down. |
| `.css` | `dracula.css.bak` and `README.md` are files in the folder and not themes. |
| 262144 bytes | a file past that is neither offered nor read. Every shipped palette is under two kilobytes. |
| Readable | a link to nothing, a file the disk refuses: not in the list. A name in the list is a theme that can be worn. |

A folder that is not there or cannot be read leaves the list holding what
ships.

The folder is watched at that one level: a theme written, saved over or taken
out of it is noticed by name as it happens.

## What a theme is called

`preset:dracula` ships inside the application; `mine:dracula` is the file
`dracula.css` in the themes folder. Both shelves may carry one filename, and
neither hides the other.

The half after the shelf is one filename in that folder.
`mine:../../../.ssh/id_rsa`, `mine:..`, `dracula` and `shelfless:dracula` name
no theme.

A name matching nothing wears `preset:numen`, and the name that was not found
is said. The settings are left as they are: putting the file back is all it
takes.

## The tokens a theme sets

Forty-two, each a colour except where it says otherwise.

### The window and what stands on it

| | |
| --- | --- |
| `--numen-surface` | the ground the window is drawn on. |
| `--numen-node-bg` | what a node, a card and a pane are filled with. |
| `--numen-node-fg` | text on that fill, and the caret in it. |
| `--numen-node-border` | the line around it. |
| `--numen-focus-bg` | the accent: the node the keyboard is on, and what a button is filled with. |
| `--numen-focus-fg` | text on the accent. |
| `--numen-focus-border` | the line around it. |
| `--numen-ring` | the outline around what is focused, and a link in marked-up text. |
| `--numen-alarm` | what is wrong: a refusal, and what the parser could not read. |

### The plex

| | |
| --- | --- |
| `--numen-seat-parent` | the hue of the seat a parent stands in. |
| `--numen-seat-child` | the hue of a child's seat. |
| `--numen-seat-jump` | the hue of a jump's seat. |
| `--numen-seat-sibling` | the hue of a sibling's seat. |
| `--numen-edge` | the line drawn between two nodes. |
| `--numen-edge-label` | the words on that line, and small print wherever it stands. |

### Text being read, and text about to change

| | |
| --- | --- |
| `--numen-selection` | the ground under selected text. |
| `--numen-selection-away` | the same selection where the keyboard is somewhere else. |
| `--numen-selection-match` | every other place that text stands. |
| `--numen-highlight` | the ground under a stretch something is about to change. It sits under words still being read, so it is a tint. |
| `--numen-caution-bg` | the band something is said in that a person should read but need not act on at once. |
| `--numen-caution-fg` | the words in that band. |

### Marked-up text

| | |
| --- | --- |
| `--numen-syntax-mark` | the marks themselves, shown under the caret. |
| `--numen-code-bg` | the ground under code. |
| `--numen-table-head-bg` | the ground under the head row of a table. |
| `--numen-syntax-keyword` | keywords, modifiers, and operators written as words. |
| `--numen-syntax-name` | names: variables, properties, attributes. |
| `--numen-syntax-type` | type names, class names, namespaces, tags. |
| `--numen-syntax-string` | strings and regular expressions. |
| `--numen-syntax-number` | numbers, booleans, atoms, escapes. |
| `--numen-syntax-comment` | comments, which are set in italic. |
| `--numen-syntax-punctuation` | operators, brackets, separators. |

### What floats over the window

| | |
| --- | --- |
| `--numen-panel-bg` | what a panel is tinted with. |
| `--numen-panel-border` | the line around it. |
| `--numen-panel-blur` | a length: how much what shows through the tint is blurred. An opaque tint leaves nothing to blur, so a theme setting one sets the other. |
| `--numen-panel-shadow` | what a panel is lifted by. It carries its offsets and its spread beside its colour, so a theme setting it writes those lengths again. |
| `--numen-shadow-card` | what a card is lifted off the window by, written the same way. |
| `--numen-scrim` | the wash a panel lays over what it covers. |

### Fields, and what is said in them

| | |
| --- | --- |
| `--numen-field-bg` | the ground a field of typing sits in. |
| `--numen-field-border` | the line around it. |
| `--numen-bubble-bg` | the bubble what was said is drawn in. |
| `--numen-bubble-fg` | the words in it. |
| `--numen-answer-fg` | what came back, set on the surface. |

## What a theme leaves alone

The type, the measures and the timings: the sans stack and the mono one, the two
text sizes and the line height, the radii and strokes, the clearances, the sizes
of a handle and a field, the order things float in, the durations and the
easing. They are the same under every palette, and `tokens.css` holds them under
the palette in three groups — what follows the interface, what the text
multiplier reaches as well, and what follows neither.

Nothing bounds a theme to the palette. A theme declaring `--numen-font-size`
declares it, and a length written in `rem` goes on following the interface.

The two multipliers are what a theme does not reach. `--numen-interface` and
`--numen-font` stand in the element the head ends with, after the mode's and the
theme's, so a theme naming either is the earlier of two declarations weighing
the same and the window is drawn at the size a person chose.

## What ships

| | |
| --- | --- |
| `preset:numen` | this product's own palette, in two halves. |
| `preset:catppuccin` | Latte in the light, Mocha in the dark. |
| `preset:github` | Light Default in the light, Dark Default in the dark. |
| `preset:gruvbox` | two halves, orange the accent. |
| `preset:solarized` | two halves, one set of accents across both. |
| `preset:ayu` | two halves, yellow the accent across both. |
| `preset:tokyo-night` | Tokyo Night in the dark, Tokyo Night Light in the light. |
| `preset:dracula` | dark, pinned. |
| `preset:nord` | dark, pinned. |
| `preset:one-dark` | dark, pinned: Atom's hues on One Dark Pro's grounds. |
| `preset:monokai` | dark, pinned: the original, as Visual Studio Code sets it. |
| `preset:cobalt2` | dark, pinned: yellow on navy. |
| `preset:amber` | dark, pinned: four colours, and the rest derived from them. |

Each is its publisher's palette from that publisher's own source; what a
palette does not name is derived from what it does, and the file says which.
Each names the colours its publisher names and leaves the rest — marked-up text
among them — as `tokens.css` has it.

The files are in `modules/apps/desktop/internal/adapter/theme/presets/`. A
first theme is one of them copied into the themes folder under a name of its
own: `mine:dracula` stands in the list beside `preset:dracula`.

## What is written down

```json
{
  "appearance": {
    "mode": "system",
    "theme": "preset:numen"
  }
}
```

Both fields are in `numen.json`, described in [Settings](settings.md).
