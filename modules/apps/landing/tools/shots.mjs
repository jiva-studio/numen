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
  { name: 'filing', story: 'application-window--filing' },
  { name: 'writing', story: 'application-window--writing' },
  {
    name: 'mapping',
    story: 'application-window--mapping',
    over: '[aria-label="The second law, child"]',
  },
  { name: 'searching', story: 'application-window--searching' },
  { name: 'asking', story: 'application-window--asking' },
  { name: 'hearing', story: 'desktop-window--recording' },
  { name: 'transcribing', story: 'desktop-window--transcribed' },
  { name: 'recognising', story: 'desktop-window--recognised' },
  { name: 'decking', story: 'desktop-window--deck' },
  { name: 'cutting', story: 'desktop-window--stencil' },
  { name: 'pacing', story: 'desktop-window--preset' },
  { name: 'owing', story: 'flash-cards-window--cards-due', size: NARROW },
  { name: 'reviewing', story: 'flash-cards-window--reviewing', size: NARROW },
]
