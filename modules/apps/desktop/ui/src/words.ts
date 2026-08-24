/**
 * What the window says in its own voice.
 *
 * A sentence about what a tab holds is said by the kind that holds it, beside
 * the code that draws it. What is left here is the window itself: its tabs,
 * the palette, the commands, the corner, and the quit.
 */
import { commandKeyWord } from '@numen/ui'
import type { Refused } from './core'
import { WORDS as agent } from './agent/words'
import { WORDS as note } from './note/words'
import { WORDS as plex } from './plex/words'

/** What the vault refused a command, in words a person reads. */
export const REFUSED: Record<Refused, string> = {
  missing: 'that note is not in the vault',
  notANote: 'that file is not a note',
  notText: 'that file is not text',
  tooLarge: 'that note is longer than this writes',
  bodyRefused: 'that text cannot be written into a note',
  unreadable: 'the frontmatter of that note cannot be read',
  occupied: 'a note of that name is filed there, so the note was renamed and its file was not',
  unnameable: 'a note cannot be called that',
}

export const WORDS = {
  newTab: 'New tab',
  /** What the window says above the work while something is wrong. */
  unwatched: 'not following the vault',
  unread: 'the vault could not be read',
  reading: 'reading the vault…',
  nothingRead: 'nothing was read',
  choose: 'What goes in this tab',
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
  notEmbedded: 'This vault has not been read for meaning yet',
  /** The commands, and the three bands they are drawn in. */
  overNote: 'This note',
  overWindow: 'This window',
  overVault: 'This vault',
  beside: 'Open beside',
  child: 'New child note',
  parent: 'New parent note',
  jump: 'New jump note',
  title: 'Change title',
  remove: 'Remove note',
  destroy: 'Destroy note',
  ask: 'Ask the agent about this note',
  copy: 'Copy path',
  newNote: note.newNote,
  newPlex: plex.newPlex,
  newAgent: agent.newAgent,
  close: 'Close this tab',
  appearance: 'Theme, and light or dark',
  findKeys: commandKeyWord(navigator.userAgent),
  first: 'Go to the note the vault opens with',
  goto: 'Go to a note',
  /** Why nothing can be done to a note: the vault is unread, or none is in front. */
  indexing: 'The vault is still being read',
  noNote: 'Nothing in front of you is a note',
  /** The steps a command asks for: the chip beside the field, and the field. */
  command: 'Command',
  typeCommand: 'Type a command',
  naming: 'Name',
  typeName: 'What is it called',
  callIt: 'Call it',
  typeNote: 'Look for a note',
  choosing: 'Appearance',
  typeChoice: 'Look for a theme',
  chooses: 'Wear it',
  /** The two shelves a theme comes off, and the one worn now. */
  shipped: 'Ships with numen',
  ownFile: 'Your own file',
  worn: 'Worn now',
  /** The three shades, and why the theme applied leaves nothing to choose. */
  system: 'Follow the system',
  light: 'Light',
  dark: 'Dark',
  pinned: 'The theme worn sets this itself',
  /** The themes could not be listed, or the one chosen could not be written. */
  unlisted: 'The themes could not be listed',
  asking: 'Confirm',
  answer: 'Choose an answer',
  keeps: 'Keep the note',
  kept: 'Nothing happens to it',
  removes: 'Remove',
  trashed: 'It goes to the .trash folder of the vault',
  exactly: 'This cannot be undone',
  typeBack: 'Type the name of the note',
  destroys: 'Destroy',
  forever: 'Nothing brings it back',
  /** The note a search did not find, offered as one to make. */
  creating: 'Nothing was found',
  creates: 'Create a note called',
  asChild: 'Create it as a child',
  asParent: 'Create it as a parent',
  asJump: 'Create it as a jump',
  /** What a command could not do, and what it left behind. */
  refused: REFUSED,
  retargeted: 'These notes link by a name that means another note now:',
  repaired: 'These notes linked by the name it had, and were written again:',
  dangling: 'These notes link to nothing now:',
  /** Where a removed note landed, which is the only way back to it. */
  trashedAt: 'The note is in the trash, at',
  /** The rename wrote in the frontmatter, which is the person's own. */
  titled: 'The title is written in the frontmatter of the note',
  nowhere: 'The vault has no note to open with',
  unanswered: 'that note changed on disk, and its tab is waiting for an answer',
  overtaken: 'that note changed on disk while this was asked, so nothing was written',
  /** What the action panel of the palette is called. */
  actions: 'Actions',
  findAction: 'Search actions',
  noAction: 'Nothing by that name',
  /** The corner where what is running behind the window is shown. */
  wordsOnly: 'Searching by words only — no model set',
  working: 'Background work',
  putAway: 'Put away',
}
