/**
 * What the window says in its own voice.
 *
 * A sentence about what a tab holds is said by the kind that holds it, beside
 * the code that draws it. What is left here is the window itself: its tabs,
 * the palette, the commands, the corner, and the quit.
 */
import { commandKeyChord, keyChord } from '@numen/ui'
import { ERRORS } from './words/errors'

export { ERRORS }

export const WORDS = {
  /** What the corner says while something about the vault is wrong. */
  unwatched: 'not following the vault',
  unread: 'the vault could not be read',
  reading: 'reading the vault…',
  nothingRead: 'nothing was read',
  going: 'These notes stopped saving because their files changed. The window waits.',
  later: 'Not yet',
  /** The palette, and the three groups it draws. */
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
  /** The commands, and the groups they are drawn in. */
  overNote: 'This note',
  overFile: 'This file',
  overWindow: 'This window',
  overVault: 'This vault',
  beside: 'Open beside',
  child: 'New child note',
  parent: 'New parent note',
  jump: 'New jump note',
  title: 'Change title',
  remove: 'Remove note',
  destroy: 'Destroy note',
  ask: 'Ask the agent',
  copy: 'Copy path',
  /** The two runs over the file in front: a recording transcribed, a scan recognised. */
  transcribe: 'Transcribe this recording',
  recognise: 'Recognise the text of this document',
  /** The transcript of the recording in front, put right by a proofreader. */
  proofread: 'Proofread the transcript of this recording',
  /** The text at the address a url points at, fetched again. */
  downloadText: 'Download the text again',
  /** What a url points at, fetched onto this disk. */
  downloadCopy: 'Download a copy',
  /** The transcript of the recording in front, taken away, and the two answers. */
  deleteText: 'Delete the transcript of this recording',
  keepsTranscript: 'Keep the transcript',
  deletes: 'Delete',
  deleted: 'The words go, and the recording can be transcribed again',
  /** The copy fetched for the url in front, taken off this disk, and its answers. */
  deleteCopy: 'Delete the copy on this disk',
  keepsCopy: 'Keep the copy',
  deletedCopy: 'The copy goes, and it can be fetched from the address again',
  reveal: 'Show this note in the files',
  /** The preset this note is: the note itself, or the one a deck is scheduled by. */
  preset: 'Open the preset',
  /** The deck in front names no preset, so the defaults schedule it. */
  noPreset: 'This deck names no preset, so it is scheduled by the defaults.',
  newNote: 'New note',
  newDeck: 'New deck',
  newStencil: 'New stencil',
  /** The note that says how the decks pointing at it are scheduled. */
  newPreset: 'New preset',
  /** The note that points at a web address. */
  importUrl: 'Import an address',
  newPlex: 'New plex',
  newAgent: 'New agent',
  files: 'Show the files of the vault',
  close: 'Close this tab',
  appearance: 'Change the theme',
  mode: 'Light or dark',
  interfaceScale: 'Interface size',
  textScale: 'Reading font size',
  syncing: 'Sync title and filename',
  hanging: 'Hang the parts of a note under its node',
  parts: 'How many parts a node hangs',
  settings: 'Settings',
  findKeys: commandKeyChord(navigator.userAgent),
  /** The commands, under the second of the two keystrokes the window keeps for itself. */
  commands: 'Show the commands',
  commandsKeys: keyChord('p', navigator.userAgent),
  first: 'Go to the note the vault opens with',
  goto: 'Go to a note',
  openVault: 'Open vault',
  newVault: 'New vault',
  newVaultDetail: 'Choose a folder',
  renameVault: 'Rename vault',
  forgetVault: 'Forget vault',
  eraseVault: 'Erase vault',
  /** Why nothing can be done to a note: the vault never opened, or none is in front. */
  noVault: 'The vault could not be opened',
  noNote: 'Nothing in front of you is a note',
  /** The steps a command asks for: the chip beside the field, and the field. */
  command: 'Command',
  typeCommand: 'Type a command',
  naming: 'Name',
  typeName: 'What is it called',
  callIt: 'Call it',
  url: 'Address',
  typeAddress: 'Paste a link',
  importIt: 'Import',
  notAnAddress: 'That is not a link a browser would open',
  typeNote: 'Look for a note',
  /** One of a list the window holds: the field, and what Enter does. */
  typeChoice: 'Choose one',
  chooses: 'Choose it',
  /** The vaults the installation holds, and why one of them cannot be chosen. */
  vaults: 'Vaults',
  typeVault: 'Look for a vault',
  gone: 'Missing',
  folder: 'Choose a folder for the vault',
  /** The list of vaults did not answer. */
  unlistedVaults: 'The vaults could not be listed',
  /** The row a list of values opens on, which is the value in force. */
  current: 'Current',
  /** The two shelves the themes are drawn in, and where a person's own go. */
  shipping: 'Ships with numen',
  owned: 'Your own themes',
  noneOwned: 'A .css file in numen/themes/, beside numen.json, is one of these',
  /** The group the three halves are drawn in. */
  half: 'Light and dark',
  /** The three halves, and why the theme worn leaves nothing to choose. */
  system: 'Follow the system',
  light: 'Light',
  dark: 'Dark',
  pinned: 'Set by the theme',
  /** The group each of the two sizes is drawn in. */
  drawing: 'How large the interface is drawn',
  setting: 'How large the text is set',
  /** The themes could not be listed, and a theme's file could not be read. */
  unlisted: 'The themes could not be listed',
  unworn: 'That theme could not be read, so it is not worn',
  /** The group the setting is drawn in, and the two it is. */
  syncingGroup: 'Sync title and filename',
  /** The group the setting is drawn in, and the two it is. */
  hangingGroup: 'Hang the parts of a note under its node',
  on: 'On',
  off: 'Off',
  /** The group the counts are drawn in. */
  partsGroup: 'How many parts stand under a node at once',
  unturned: 'That setting could not be written:',
  /** The settings the vault answered with could not be read. */
  unreadSettings: 'The settings could not be read, so what is drawn is what they last held.',
  asking: 'Confirm',
  several: (files: number) => `${files} files`,
  answer: 'Choose an answer',
  kept: 'Nothing happens to it',
  keepsVault: 'Keep the vault',
  forgets: 'Forget',
  stays: 'The folder stays where it is',
  exactly: 'This cannot be undone',
  typeBack: 'Type the name of the note',
  destroys: 'Destroy',
  forever: 'Nothing brings it back',
  typeVaultBack: 'Type the name of the vault',
  erases: 'Erase',
  binned: 'The folder goes to the trash this machine keeps',
  /** The note a search did not find, offered as one to make. */
  creating: 'Nothing was found',
  creates: 'Create a note called',
  asChild: 'Create it as a child',
  asParent: 'Create it as a parent',
  asJump: 'Create it as a jump',
  errors: ERRORS,
  dangling: 'These notes link to nothing now:',
  /** The vault opens with no note at all. */
  nowhere: 'The vault has no note to open with',
  unanswered: 'that note changed on disk, and its tab is waiting for an answer',
  stale: 'that note changed on disk while this was asked, so nothing was written',
  /** A file landed where something of its name is filed, and stayed where it was. */
  occupied: 'something of that name is filed there, so the file stayed where it was',
  /** This build cannot do the run at all, and stops offering it. */
  unrunnable: 'this installation of numen cannot do that at all',
  /** What the action panel of the palette is called. */
  actions: 'Actions',
  findAction: 'Search actions',
  noAction: 'Nothing by that name',
  /** The corner where what is running behind the window is shown. */
  wordsOnly: 'Searching by words only — no model set',
  isWorking: 'Background work',
  dismiss: 'Put away',
  more: 'more',
}
