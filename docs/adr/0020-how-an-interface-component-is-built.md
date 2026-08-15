# ADR-0020: How an interface component is built

- **Status:** Accepted
- **Date:** 2026-08-15
- **Applies to:** `modules/libs/ui`
- **Related:** ADR-0000, ADR-0014, ADR-0016, ADR-0017

## Context

ADR-0014 settled what the core is and named the desktop shell as its second
entry point, and said explicitly that no decision about the interface was being
made there. This is that decision.

It is taken now because the first component is the plex — the focused
neighbourhood view the product is named for — and because it is the hardest one
this repository will contain. It moves, it is read by eye rather than by
reading, and it draws things whose meaning is decided somewhere else entirely.
Whatever rules survive it will hold for a button.

Two facts make the shape of the answer non-obvious.

The first is that **the vault format is not settled**. ADR-0003 has not been
written, so what a link is, what roles it carries, and what a note is called by
are all open. A component built against today's guess at that would be rewritten
when the guess is replaced.

The second is that **an interface is the part of a system that is hardest to
test**. A scan can be checked by reading rows out of a table. A drawing cannot,
and the usual answer — screenshots, or a browser driving a real page — is slow,
flaky, and silent about everything except the pixels it happened to capture.

## Decision

### Vue 3 with TypeScript, in `modules/libs/ui`

Vue, for the reason ADR-0014 gives for Wails: the shell is a webview, and Vue's
reactivity fits a view whose whole job is to re-derive a picture from a value
that changed. TypeScript in strict mode, with `exactOptionalPropertyTypes` and
`noUncheckedIndexedAccess`, because a component's props are a contract with
another module and an unchecked one is not a contract.

The module exists at all — against ADR-0014's rule that `modules/libs/` stays
empty until two applications genuinely share code — because the mobile client is
a webview too. That is the second consumer, and it is why the boundary is drawn
now rather than discovered later by copying a file.

### A component knows nothing about the domain

**This is the rule the module exists to keep, and every other rule here is
downstream of it.**

No component may import from `modules/libs/domain`, from the wire types, or from
anything that knows what a vault is. Not in the components, not in the stories,
not in the fixtures — a fixture taken from the domain is how the dependency
comes back in through the door marked "tests", and once the stories read it the
module is no longer independent of anything.

A component takes props it defines itself, in a vocabulary about drawing, and
emits events carrying **opaque identifiers**. Whoever renders it translates the
domain into that shape and translates the identifier back. So the plex speaks of
*a node with a role*, never of a note with links; it has no way to ask what any
of them mean, and it does not need one.

Three things follow, and they are the point:

- **ADR-0003 cannot break a component.** When what a link is finally gets
  decided, the adapter in the application changes and nothing here does.
- **A component cannot answer a question about the graph.** The plex does not
  work out who is a sibling of whom — that is a rule about relationships, it
  belongs with whoever knows the relationships, and a component that derived it
  would know the domain by definition.
- **The dependency runs one way.** `modules/apps/*` depends on
  `modules/libs/ui`. Never the reverse, and never sideways.

### The view is humble; the decisions are values

Every component splits in two, along the line Michael Feathers drew in 2002 and
Fowler wrote up as the humble dialog box.

**The core is pure.** Given the props, it computes what is to be drawn — every
coordinate, every state, every derived flag — as plain values, with no DOM, no
Vue, no clock, no randomness and no measurement of text. The same input gives
the same numbers on any machine on any day.

**The view is humble.** It receives those values and turns them into elements.
It works nothing out. If a number appears in the template that the core did not
produce, the split has been broken.

This is not layering for its own sake, and it is not the same argument ADR-0014
makes about ports. It is that **the interesting behaviour of an interface is
otherwise unreachable.** "What does the plex look like forty per cent of the way
through a move" is a question with an answer; made pure, that answer is a value
with a test, and made impure it is a moment that has to be caught with a
screenshot. Ninety-odd assertions about the plex run in under a second and not
one of them opens a browser.

**Anything with a lifetime is reached through a port**, exactly as in the core:
the clock, `requestAnimationFrame`, the size of the window, the reader's motion
preference. Each is a parameter with a browser-shaped default, so the component
works with no ceremony in an application and is fully determined in a test.

### A concept is declared once

A role, a state, a variant — anything the component enumerates — is declared in
**one** place, and everything that varies with it reads from there: the type
union, the layout, the wording, the colour.

The rule exists because the alternative was measured. Adding one role to the
plex used to mean six edits across six files, and the compiler could only see
three of them. That is the shape ADR-0014 rejects for aggregates — *the unit of
organisation is the thing, not the kind of thing* — and it applies to a set of
variants exactly as it applies to a folder of SQL.

### What a component may be extended by is an interface, not a flag

A component that will be asked to do a second thing takes the second thing as a
function, not as a `mode` prop and a branch. The plex arranges a neighbourhood
by handing the nodes to a placement strategy; rows-and-columns is the one that
ships, and a radial mind map is another implementation of the same signature
rather than an `if` inside the first.

This is a **narrow** exception to ADR-0016's rule that a thing is built with the
code that reads it rather than in anticipation of code that might. It is taken
only where a second implementation is actually intended, and it is cheap here
because the strategy is a pure function: a second one is testable on its own
terms without a component existing to host it.

It is not licence to make every decision pluggable. A knob with one setting is a
knob nobody has read, and it costs a signature that has to be honoured forever.

### Styling is design tokens, and only design tokens

Every colour, size, radius and duration a component paints with is a CSS custom
property, declared in one file. A component never reaches past a token for a
value.

This is what keeps the choice of a general component library open. The plex is
drawn from tokens and from nothing else, so whichever library is eventually
picked for buttons and dialogs sets the tokens and the plex follows without
being touched. Tokens are theme-aware by construction; a component that hard-codes
a colour has decided the theme for the whole application.

### Storybook is where a component is built

A component is written, looked at and judged in Storybook, in isolation, before
any application renders it. Not because a gallery is nice to have, but because a
component built inside a screen is a component whose edge cases are whatever
that screen happened to contain.

**The stories are the corpus, and they are awkward on purpose.** The rule is
ADR-0017's, unchanged: a fixture of three tidy items tests the component against
its own assumptions. So a component ships stories for the empty case, the single
case, the far too many case, text that is not Latin, text far too long, text with
nothing to break at, and no text at all.

**A setting is a control, not a story.** Anything reachable by turning a knob —
a duration, a density, a toggle — is an `argType`, and a second story that
differs only by the value of one is deleted.

### How a component is tested

Three levels, in descending order of how much they are worth.

1. **The core, as plain functions.** No DOM. This is where the assertions that
   matter live, because this is where the decisions are.
2. **The behaviour, with `@vue/test-utils` in jsdom.** What is emitted, what is
   reachable by keyboard, what a screen reader is told. **The negatives belong
   here** — what the component does *not* emit, what is *not* focusable — because
   those are what fail silently and look right in every screenshot.
3. **The stories, run as tests.** Each story is rendered in a browser, so a
   story that stops rendering is a failing test rather than a surprise, and the
   fixture is written once instead of twice.

**ADR-0017's rule carries over unchanged and is the important one: a test must
be able to fail.** Break the rule, watch the test fail, put it back. It matters
more here than in the core, because "the nodes were arranged" and "the component
rendered" pass under almost any implementation — including a wrong one.

Accessibility is part of the contract, not a later pass: reachable by keyboard,
named for a screen reader, honouring `prefers-reduced-motion`. It is asserted at
level 2 and therefore cannot quietly rot.

### What is deliberately not decided

**Which general component library to use.** shadcn-vue, PrimeVue, a headless
core, or none — the question is open, and it stays open until a screen exists
that needs a dialog, a menu or a table. The product is not mostly made of those:
it is a plex, an editor, a result list and a review card. Choosing a library
before there is anything to build with it means inheriting a design language to
suit components that may never be used.

The token rule above is what makes deferring free.

## Consequences

**Positive**

- The interface can be built now, while the vault format is still open, and none
  of it is invalidated when ADR-0003 lands.
- The part of an interface that normally cannot be tested — arrangement, state,
  movement — is tested as values, quickly and without a browser.
- A second renderer is one adapter. Nothing in the core knows it is SVG.
- The general component library can be chosen late, on evidence, from a position
  where switching costs a stylesheet.

**Negative**

- The humble split is visible before it pays. A component that only ever draws
  its props unchanged carries a core that does nothing, and the indirection
  reads as ceremony until the first thing that has to be derived arrives.
- Refusing domain types means a translation layer in every application that
  renders a component, and the same mapping written twice if desktop and mobile
  do not share theirs. That is the price of the boundary, and it is the price
  ADR-0014 already accepted once for Go modules.
- Storybook is a second toolchain to keep alive, and a story is code that ships
  to nobody.
- A strategy taken as an interface is a signature that has to be honoured even
  if the second implementation never arrives.

## Alternatives considered

**Components that take domain types directly.** Rejected. It is the cheaper
thing to write and the reason the vault format could not then be changed: every
component becomes a consumer of a format that ADR-0003 has not settled, and the
module ends up versioned against the domain it was extracted to be independent
of.

**Testing the interface through the browser only** — Playwright, screenshots,
visual diffs. Rejected as the primary method, kept as level 3. It is slow enough
to be run rarely, silent about everything it did not photograph, and it fails on
a font substitution as loudly as on a real defect. The reason the plex has
assertions about a frame halfway through a movement is that the frame is a
value; through a browser it is a moment nobody would have tried to catch.

**Adopting a component library first and building the plex inside it.**
Rejected: the plex has nothing in common with anything such a library ships, so
the library would be chosen on the strength of components that are not yet
needed, and its design language would be inherited before there was anything to
judge it against.

**Splitting the plex into a component per node and per edge.** Rejected: a node
is not reused anywhere on its own, so the split buys nothing and costs prop
drilling through a tree that exists only to be drawn.
