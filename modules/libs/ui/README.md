# @numen/ui

Interface components shared by the desktop and mobile clients.

**The rules are [ADR-0020](../../../docs/adr/0020-how-an-interface-component-is-built.md).**
What follows is where they live in this tree.

## The one rule

**A component here knows nothing about the domain.** No import from
the application's domain, no wire types, no vault vocabulary — not in the
components, not in the stories, not in the fixtures. A component takes props it
defines itself and emits events carrying opaque identifiers; whoever renders it
translates the domain into that shape.

This is why the plex talks about *nodes with a seat* rather than about notes
with links. What a link is is decided elsewhere; when it changes, the adapter in
the application changes and nothing in this module does.

The dependency runs one way: `modules/apps/*` depends on `modules/libs/ui`.
Never the reverse, and never sideways.

## Working on it

```bash
npm install
npm run storybook     # http://localhost:6006 — every component, every state
npm test
npm run typecheck
```

Storybook is the development environment. A component is built and judged
there, in isolation, before any application renders it.

## Layout

```
src/
  tokens/            the whole styling contract
    tokens.css       every value, as custom properties
    theme.css        Tailwind, told what those values are
  lib/utils.ts       cn(), for joining class lists
  components/ui/     what shadcn-vue supplies, and this module now owns
  fixtures/          awkward text, and a backdrop for a panel to float over
  panel/             the floating panel: a surface to put things on
  composer/          the field a message is written in
    model.ts         what state it is in, and what a key means
  dots/              three dots rising in turn: something is being written
  thread/            the conversation
    model.ts         every voice, declared once — a new voice starts here
  assembled/         the three of them in one piece, for Storybook
  plex/              the focused-neighbourhood view
    model/           what a plex is made of, as plain values
      seat.ts          every seat, declared once — a new seat starts here
      node.ts          a node, placed or not; what is true of one wherever
                       it is drawn — its name, whether it can be chosen,
                       where its handle sits
      edge.ts          an edge, routed or not
      neighbourhood.ts the input, and the two things it must be true about
      frame.ts         everything to be drawn, at one moment
    arrange/         the decisions — no DOM, no Vue, no clock
      options.ts       sizes, limits and routing, as a value object
      placement.ts     WHERE the nodes go — a strategy, replaceable
      routing.ts       HOW a line runs between two boxes
      arrange.ts       admit, place, route
      interpolate.ts   one frame between two arrangements
    render/          the drawing, and nothing else
      PlexView.vue     the picture between the nodes: the window, the edges,
                       and the gesture crossing them
      PlexNodeView.vue one node: its box, its title, its handle
    transition.ts    the clock, behind a port
    Plex.vue         the composition root
```

**The core is pure and the view is humble.** Every number in the template came
from `arrange/`; if one appears there that did not, the split is broken. That
split is what makes "what does the plex look like forty per cent of the way
through a move" a value with a test rather than a screenshot to be caught.

**Two seams are meant to be used.** `Placement` decides coordinates and nothing
else — a radial mind map is another implementation, not a branch inside this
one. `Environment` is the clock, so a movement is stepped by hand in a test and
by the browser everywhere else.

## Styling

**Two ways of writing a style, along one line.** A component made of DOM is
styled with Tailwind utilities. The plex is styled with scoped CSS, because
what it paints with — `rx`, `stroke-dasharray`, `paint-order`, `r` — are SVG
attributes that no utility expresses.

Both end at the same place: `tokens.css` holds every value, `theme.css` defines
Tailwind's names in terms of those tokens, and `bg-surface` and
`var(--numen-surface)` are one value.

**A theme is chosen by `color-scheme`.** The tokens are `light-dark()` pairs,
and that is the whole mechanism — Storybook's switch sets `color-scheme` and
everything follows. It is declared on `:root` and **nowhere else**: an element
that declares it again hands the choice back to the reader's system, and every
token under that element resolves to the wrong half of its pair while the page
looks like it asked for the other one. **No component writes a `dark:` class.**
Tailwind's `dark:`
variant reads a class or the OS setting, neither of which is what is being
switched, so a component using one is stuck in whichever theme it was written
in. Components taken from shadcn-vue have theirs removed on the way in.

**Scoped CSS beats a utility.** Component styles are unlayered and Tailwind's
are in `@layer`, so an unlayered rule wins whatever its specificity. A scoped
block on a DOM component is therefore for what a utility cannot say — the
composer's field and its copy sharing one grid cell — and never for what one
can.

**A component root carries `numen`.** The tokens are declared on `:root` and on
`.numen` both, so a component works in a page that never set them.

## Reaching out from a node

Every node offers a handle when a pointer is over it. Dragging from it and
letting go on empty space asks for a new node; letting go on another node asks
for a link between the two. Which seat either lands in comes from the direction
the gesture went, read off the arrangement's own `direction` — so inverting the
plex inverts the gesture with it. Escape gives up; a click without travel asks
for one more child, which is what anyone reaches for.

**The seat is decided when the gesture ends, not when it begins.** TheBrain
does the opposite: it has a gate per seat, and which one you grab settles it.
Deciding at the end means the reader cannot know what they will get, which is
why the gesture draws the seat it would land in as it goes. Recorded here
rather than in an ADR because it is a decision about an interaction, not about
how components are built.

The plex reports and does not act: `create` and `link` carry identifiers and a
seat, and what a parent *means* — what gets written, whether it is allowed —
belongs to whoever answers. A sibling is another of the parent's children
rather than something anyone makes directly, so it is left out of `creatable`
by default; that default is the caller's to change.
