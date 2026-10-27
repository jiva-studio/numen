/**
 * What the page shows, one to a band and one over them all.
 *
 * The list stands on its own so that reading it costs nothing: the check that
 * every name here is still a story runs on a bare node, and a camera nobody
 * has installed is no reason for the names to go unchecked.
 *
 * `over` names what the pointer is left on: the picture is taken once whatever
 * a hand resting there brings out is all the way out. `size` is the window it
 * is drawn in, and the wide one where it names none. `drawn` is the fewest
 * bytes the picture comes to, for a screen that is mostly empty.
 */

/** The window a person runs their cards in: standing, and not a wide window. */
export const NARROW = { width: 760, height: 940 }

export const SHOTS = [
  { name: 'files', story: 'application-window--filing' },
  { name: 'editor', story: 'application-window--writing' },
  {
    name: 'graph',
    story: 'plex-map--mapping',
    over: '[aria-label="The second law, child"]',
  },
  { name: 'search', story: 'application-window--searching' },
  { name: 'agent', story: 'application-window--asking' },
  { name: 'audio', story: 'recording-tab--playing' },
  { name: 'transcript', story: 'recording-tab--no-transcript', drawn: 5_000 },
  { name: 'ocr', story: 'application-window--recognising' },
  { name: 'deck', story: 'flashcards-deck--deck' },
  { name: 'card-model', story: 'flashcards-card-model--card-model' },
  { name: 'preset', story: 'window-preset--a-preset' },
  { name: 'cards-due', story: 'flash-cards-window--cards-due', size: NARROW },
  { name: 'review', story: 'flash-cards-window--reviewing', size: NARROW },
]
