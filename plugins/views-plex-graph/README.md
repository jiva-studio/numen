# Plex Graph Visualizer Plugin

Knowledge graph visualization and plex explorer plugin for Numen.

## Features
- Interactive 2D knowledge graph visualization of vault notes and relationships
- Quick link creation popover
- Link inspector for bi-directional link relationships

## The graph

`frontend/plex/` draws a neighbourhood of notes. The plex talks about *nodes with a relation*: a link is not what the plex draws. What a link is is decided elsewhere; when it changes, the adapter in the plugin changes and nothing under `frontend/plex/` does.

```
plex/              the focused-neighbourhood view
  model/           what a plex is made of, as plain values
    relation.ts          every relation, declared once — a new relation starts here
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

**The core is pure and the view is humble.** Every number in the template came from `arrange/`; if one appears there that did not, the split is broken. That split is what makes "what does the plex look like forty per cent of the way through a move" a value with a test and not a screenshot to be caught.

**Two seams are meant to be used.** `Placement` decides coordinates and nothing else — a radial mind map is another implementation, not a branch inside this one. `Clock` is the clock, so a movement is stepped by hand in a test and by the browser everywhere else.

### Reaching out from a node

Every node offers a handle when a pointer is over it. Dragging from it and letting go on empty space asks for a new node; letting go on another node asks for a link between the two. Which relation either lands in comes from the direction the gesture went, read off the arrangement's own `direction` — so inverting the plex inverts the gesture with it. Escape gives up; a click without travel asks for one more child, which is what anyone reaches for.

**The relation is decided when the gesture ends, not when it begins.** TheBrain does the opposite: it has a gate per relation, and which one you grab settles it. Deciding at the end means the reader cannot know what they will get, which is why the gesture draws the relation it would land in as it goes.

The plex reports and does not act: `create` and `link` carry identifiers and a relation, and what a parent *means* — what gets written, whether it is allowed — belongs to whoever answers. A sibling is another of the parent's children rather than something anyone makes directly, so it is left out of `creatable` by default; that default is the caller's to change.
