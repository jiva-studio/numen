/**
 * What the page shows, one to a band and one over them all.
 *
 * The list stands on its own so that reading it costs nothing: the check that
 * every name here is still a story runs on a bare node, and a camera nobody
 * has installed is no reason for the names to go unchecked.
 *
 * `over` names what the pointer is left on: the picture is taken once whatever
 * a hand resting there brings out is all the way out. `size` is the window it
 * is drawn in, and the wide one where it names none.
 */

/** The window a person runs their cards in: standing, and not a wide window. */
export const NARROW = { width: 760, height: 940 }

export const SHOTS = [
  { name: 'files', story: 'application-window--filing' },
  { name: 'editor', story: 'application-window--writing' },
  {
    name: 'graph',
    story: 'application-window--mapping',
    over: '[aria-label="The second law, child"]',
  },
  { name: 'search', story: 'application-window--searching' },
  { name: 'agent', story: 'application-window--asking' },
  { name: 'audio', story: 'desktop-window--recording' },
  { name: 'transcript', story: 'desktop-window--transcribed' },
  { name: 'ocr', story: 'desktop-window--recognised' },
  { name: 'deck', story: 'desktop-window--deck' },
  { name: 'stencil', story: 'desktop-window--stencil' },
  { name: 'preset', story: 'desktop-window--preset' },
  { name: 'cards-due', story: 'flash-cards-window--cards-due', size: NARROW },
  { name: 'review', story: 'flash-cards-window--reviewing', size: NARROW },
]
