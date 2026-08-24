/**
 * Everything the window says in its own voice.
 *
 * One sentence is written once and read by whoever draws it. What a module of
 * its own says — the palette, the corner, a talk — is asked for through the
 * `Words` each of them declares, and this answers all of them.
 */
export const WORDS = {
  ask: 'Ask about this note',
  thinking: 'Thinking',
  unreachable: 'The agent could not be reached.',
  nothing: 'The agent finished without saying anything.',
  unsent: 'Did not send',
  send: 'Send',
  stop: 'Stop the agent',
  newTab: 'New tab',
  choose: 'What goes in this tab',
  newNote: 'New note',
  newPlex: 'New plex',
  newAgent: 'New agent',
  plex: 'Plex',
  agent: 'Agent',
  stopped: 'The agent stopped here',
  nothingSaid: 'Nothing said yet',
  words: 'Searching by words only — no model set',
  overtaken: 'The file changed on disk, so this note stopped saving.',
  gone: 'This note is no longer in the vault, so saving stopped. What is here is still yours.',
  makeAgain: 'make it again',
  keep: 'Keep mine',
  take: "Take the file's",
  going: 'These notes stopped saving because their files changed. The window waits.',
  later: 'Not yet',
  /** The palette, and the three bands it draws. */
  find: 'Search the vault',
  names: 'Names',
  text: 'Text',
  meaning: 'Meaning',
  travel: 'Show in plex',
  read: 'Open the note',
  readAt: 'Open at this heading',
  readDocument: 'Open the document here',
  noneFound: 'Nothing',
  notAsked: 'The vault could not answer',
  typeToFind: 'Type to look for a note',
  /** The corner where what is running behind the window is shown. */
  working: 'Background work',
  putAway: 'Put away',
  /** A document read: turning its pages, and how close it is drawn. */
  back: 'Previous page',
  next: 'Next page',
  page: 'Page',
  closer: 'Closer',
  further: 'Further',
}
