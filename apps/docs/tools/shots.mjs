/**
 * What the manual shows, and from which story.
 *
 * The list stands on its own so that reading it costs nothing: the check that
 * every name here is still a story runs on a bare node, and a camera nobody
 * has installed is no reason for the names to go unchecked.
 *
 * A window is drawn narrower than the column it lands in, so the application's
 * own text is larger on the page than it is on screen and can be read at a
 * glance. What stands alone is drawn at about the size it stands at.
 *
 * `over` names what the pointer is left on: the picture is taken once whatever
 * a hand resting there brings out is all the way out. `preset` names a palette
 * that ships with the application, which the window is drawn wearing. `drawn`
 * is the fewest bytes the picture comes to, for a screen that is mostly empty.
 */
export const SHOTS = [
  { name: 'window', story: 'application-window--plex-map', width: 1180, height: 740 },
  { name: 'theme-dracula', story: 'application-window--plex-map', width: 1180, height: 740, preset: 'dracula' },
  { name: 'theme-solarized', story: 'application-window--plex-map', width: 1180, height: 740, preset: 'solarized' },
  { name: 'searching', story: 'application-window--searching', width: 1180, height: 740 },
  { name: 'asking', story: 'application-window--asking', width: 1180, height: 740 },
  { name: 'plex', story: 'application-window--mapping', width: 1180, height: 740 },
  {
    name: 'parts',
    story: 'application-window--nesting',
    width: 1180,
    height: 740,
    over: '.plex__node--focus',
  },
  { name: 'commands', story: 'application-window--commanding', width: 1180, height: 740 },
  { name: 'writing', story: 'application-window--writing', width: 1180, height: 740 },
  { name: 'table', story: 'application-window--tabling', width: 1180, height: 740 },
  { name: 'reader', story: 'application-window--reading', width: 1180, height: 740 },
  { name: 'files', story: 'application-window--filing', width: 1180, height: 740 },
  { name: 'settings', story: 'desktop-window--settings', width: 1180, height: 740 },
  { name: 'recording', story: 'recording-tab--playing', width: 1180, height: 740 },
  {
    name: 'transcribe',
    story: 'recording-tab--no-transcript',
    width: 1180,
    height: 740,
    drawn: 5_000,
  },
  { name: 'recognise', story: 'application-window--recognising', width: 1180, height: 740 },
  { name: 'deck', story: 'flashcards-deck--deck', width: 1180, height: 740 },
  { name: 'stencil', story: 'flashcards-stencil--stencil', width: 1180, height: 740 },
  { name: 'preset', story: 'window-preset--a-preset', width: 1180, height: 740 },
  // The window a person runs their cards in, which is not a wide window.
  { name: 'decks', story: 'flash-cards-window--cards-due', width: 760, height: 540 },
  { name: 'session', story: 'flash-cards-window--reviewing', width: 760, height: 540 },
]
