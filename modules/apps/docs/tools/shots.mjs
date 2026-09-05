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
 * that ships with the application, which the window is drawn wearing.
 */
export const SHOTS = [
  { name: 'window', story: 'application-window--map', width: 1180, height: 740 },
  { name: 'theme-dracula', story: 'application-window--map', width: 1180, height: 740, preset: 'dracula' },
  { name: 'theme-solarized', story: 'application-window--map', width: 1180, height: 740, preset: 'solarized' },
  { name: 'searching', story: 'application-window--searching', width: 1180, height: 740 },
  { name: 'asking', story: 'application-window--asking', width: 1180, height: 740 },
  { name: 'plex', story: 'application-window--mapping', width: 1180, height: 740 },
  {
    name: 'parts',
    story: 'application-window--hanging',
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
  { name: 'recording', story: 'desktop-window--recording', width: 1180, height: 740 },
  { name: 'transcribe', story: 'desktop-window--transcribed', width: 1180, height: 740 },
  { name: 'recognise', story: 'desktop-window--recognised', width: 1180, height: 740 },
  { name: 'deck', story: 'desktop-window--deck', width: 1180, height: 740 },
  { name: 'stencil', story: 'desktop-window--stencil', width: 1180, height: 740 },
  { name: 'preset', story: 'desktop-window--preset', width: 1180, height: 740 },
  // The window a person runs their cards in, which is not a wide window.
  { name: 'decks', story: 'flash-cards-window--cards-due', width: 760, height: 540 },
  { name: 'sitting', story: 'flash-cards-window--reviewing', width: 760, height: 540 },
]
