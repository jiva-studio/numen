# ADR-0043: How large the interface is drawn, and how large the text is set

- **Status:** Proposed
- **Date:** 2026-08-25
- **Applies to:** `modules/libs/ui`, `modules/apps/desktop`,
  `modules/libs/protocol`
- **Partly supersedes:** ADR-0013 — the settings file, for what the application
  writes into it
- **Related:** ADR-0013, ADR-0017, ADR-0020, ADR-0024, ADR-0025, ADR-0028,
  ADR-0035, ADR-0037, ADR-0041

## Context

Nothing about size is a setting a person can change while the window is open.
`appearance.zoom` is read once, at `cmd/numen/main.go`, and handed to the window
as it is made.

That zoom is the toolkit's own, and it does less than it appears to. Every
backend raises a value under 1 to 1, so it cannot make anything smaller. macOS
never applies it: the line that would is commented out in the toolkit. Windows
applies it at construction without the raise, so one platform honours a value
the other two discard.

Underneath, six text sizes are drawn in one window and four are measured from
the page's root rather than from anything this product names:

| | | measured from |
| --- | --- | --- |
| `text-base`, 13px | `--numen-font-size` | this product |
| `text-small`, 10px | `--numen-edge-label-size` | this product |
| `0.8rem`, 12.8px | `NoteTab.vue`, `Trouble.vue` | the page's root |
| `0.85rem`, 13.6px | `BlankTab.vue`, `Leaving.vue` | the page's root |
| `0.9rem`, 14.4px | `Trouble.vue` | the page's root |
| `0.875rem`, 14px | `prose-sm` on `Prose.vue` | the page's root |

The last is what a person reads. `prose-sm` sets the size of marked-up text and
a scale above it for headings, code, lists and quotations, and no token reaches
it.

The plex is a seventh case. Its geometry is numbers in `arrange/options.ts`, in
CSS pixels, and its node labels are set from `--numen-font-size`. A label that
grows inside a box that does not is a label that ellipsises.

## Decision

### Two settings, because a person means two things

**`appearance.interface`** is how large the interface is drawn: chrome,
controls, spacing, panels, and the type in them. A palette row, a menu item, a
tab and a button grow together, because that is what a larger interface is.

**`appearance.font`** is how large the text a person reads is set: a note, a
book, an answer, the editor.

Each is a multiplier, 1 being as designed.

A settings file naming `zoom` names `interface`. The two say the same thing, so
the value carries over and a window drawn at 1.5 goes on being drawn at 1.5.

A file that names no size, and `zoom: 0` names none, is drawn at what the
desktop asks for: `GDK_DPI_SCALE` is how large a session has its text scaled,
and the interface is drawn to match. A number outside the range is not seeded.

### The interface size is the root's font size, and 1 changes nothing

```css
:where(:root) { font-size: calc(16px * var(--numen-interface, 1)); }
```

A length in `rem` is already a multiple of 16px everywhere in this product and
in the toolkit beneath it. Multiplying the root by the setting carries all of
them — Tailwind's spacing scale, the `rem` lengths components hold, the radii —
and at 1 every one of them resolves to the pixel it resolves to today.

The toolkit's own zoom goes. It cannot shrink, and it is applied on one platform
of three.

### `zoom` is the reader's word again

With the window's zoom gone, `zoom` means one thing: how large a page of a
document is drawn. ADR-0037 already uses it that way.

### Two text sizes, because the window holds two kinds of text

The chrome's type is the interface's: `--numen-font-size` is written in `rem`
and the root carries it, with the lengths that surround it — the height of a
field and a button, the clearance inside a node, the inset typing keeps from the
end of its field. A theme declaring `--numen-font-size` declares it.

The text a person reads has a token of its own, and `appearance.font` is what
multiplies it. `prose-sm` states its base in `rem` and everything above it in
`em`, so stating that base from this token brings the whole scale onto it. The
editor is set from it too.

### What follows which, and what follows neither

A hairline, a border, a focus ring and the stroke of a handle follow neither.
They are one physical line and they stay in `px`.

Each token in `tokens.css` is placed under one of these three, in the file,
beside its value.

### The plex is told its size

`Plex.vue` already takes `options` as a prop and merges them with the viewport
it measures. The window computes that option set from the two multipliers and
passes it down: what holds a label from the text, what separates nodes from the
interface.

`arrange/` is not touched. It reads no clock, measures no text and touches no
DOM, and it goes on receiving numbers.

### Both are changed while the window is open, and it opens wearing them

They join the mode and the theme in the head of the served page, so no frame is
drawn at a size nobody asked for. A size arriving after the first frame is a
reflow, which is more than the repaint a colour would cost.

Each is a command in the band over the window. What the keyboard lands on is
applied, and leaving without choosing puts back what was there.

**The preview is held.** A theme is a stylesheet swapped. A size relays out the
document, and in a document window it shifts every page into another width and
empties what the core has drawn. The reader already holds a resize for this
reason and the same hold applies here.

### Where the range ends, and how a person gets back

Both settings are bounded, and a number outside the bound is refused and said.

A window can still be made hard to read, and the palette is drawn at whatever
was chosen. `-interface` and `-font` say it for one launch, over the file, the
way `-zoom` did.

## Consequences

**Positive**

- One number moves the interface and another moves the text, which is what a
  person means by each.
- At 1 nothing moves. The change is inert until it is asked for.
- Shrinking works, on every platform, which the toolkit's zoom never did.

**Negative**

- The toolkit's zoom scaled the pictures a document's pages are drawn as, and a
  CSS interface scale does not. A person whose `zoom: 1.5` carried over sees
  their pages at the size they were while everything around them grew, and the
  reader's own zoom is what moves them.
- Three kinds of length now live in `tokens.css` where there were two, and every
  token has to be placed under one of them by hand.
- A person can still make the window hard to read, and the way back is a flag or
  the settings file.

## Alternatives considered

**Keeping the toolkit's zoom for the interface.** Rejected. It cannot make
anything smaller, and it is applied on one platform of three.

**The root's font size taken from `--numen-font-size`.** Rejected. It ties the
interface to the text, and because `rem` is 16px everywhere beneath this product
it would draw every screen 18.75% smaller the first time it was switched on,
with nothing asked for.

**One setting for both.** Rejected. It is what the toolkit's zoom already was.

**The plex exempt, its labels fixed in px.** Rejected. The plex is the view this
product is named for.
