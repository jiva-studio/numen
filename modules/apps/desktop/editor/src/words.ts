/**
 * What the window says in its own voice.
 *
 * A sentence about what a tab holds is said by the kind that holds it, beside
 * the code that draws it. What is left here is the window itself: its tabs,
 * the palette, the commands, the corner, and the quit.
 */
import { commandKeyChord, keyChord } from '@numen/ui'
import type { Artifact, ArtifactState, RefusalReason, VaultRefusalReason } from './core'
import { WORDS as agent } from './agent/words'
import { WORDS as cards } from './cards/words'
import { WORDS as note } from './note/words'
import { WORDS as plex } from './plex/words'
import { WORDS as preset } from './preset/words'

/** What the vault refused a command, in words a person reads. */
export const REFUSED: Record<RefusalReason, string> = {
  missing: 'that note is not in the vault',
  notANote: 'that file is not a note',
  notText: 'that file is not text',
  tooLarge: 'that note is longer than this writes',
  bodyRefused: 'that text cannot be written into a note',
  unreadable: 'the frontmatter of that note cannot be read',
  occupied: 'a note of that name is filed there, so the note was renamed and its file was not',
  unnameable: 'a note cannot be called that',
  notAStencil: 'that note is not a stencil',
  notADeck: 'that note is not a deck',
  deckTooLarge: 'that deck is longer than this reads',
  notAPreset: 'that note is not a preset',
}

/**
 * What an artifact of a file now stands at, in the window's own voice.
 *
 * They answer a person who chose Recognise, Transcribe or Proofread from a
 * menu, and they say the word that person chose. A run under way is said as a
 * report and everything else as a refusal, so no two of them may say the same
 * thing.
 */
export const MADE: Record<Artifact, Record<ArtifactState, string>> = {
  ocr: {
    none: 'This scan has not been recognised.',
    queued: 'This scan is in line, behind the one being recognised now.',
    running: 'This scan is being recognised now.',
    stopped: 'Recognising this scan stopped part way.',
    done: 'This scan has already been recognised.',
    empty: 'Nothing was read in this scan.',
    failed: 'This scan could not be opened:',
  },
  transcript: {
    none: 'This recording has not been transcribed.',
    queued: 'This recording is in line, behind the one being transcribed now.',
    running: 'This recording is being transcribed now.',
    stopped: 'Transcribing this recording stopped part way.',
    done: 'This recording has already been transcribed.',
    empty: 'No speech was heard in this recording.',
    failed: 'This recording could not be opened:',
  },
  'transcript.corrected': {
    none: 'Nothing has been transcribed here, so there is nothing to proofread.',
    queued: 'This transcript is in line, behind the one being put right now.',
    running: 'This transcript is being put right now.',
    stopped: 'Putting this transcript right stopped part way.',
    done: 'This transcript has already been put right.',
    empty: 'There were no words in this transcript to put right.',
    failed: 'This transcript could not be put right:',
  },
  'ocr.corrected': {
    none: 'Nothing has been recognised here, so there is nothing to proofread.',
    queued: 'This reading is in line, behind the one being put right now.',
    running: 'This reading is being put right now.',
    stopped: 'Putting this reading right stopped part way.',
    done: 'This reading has already been put right.',
    empty: 'There were no words in this reading to put right.',
    failed: 'This reading could not be put right:',
  },
  copy: {
    none: 'No copy of this video is on this disk.',
    queued: 'This video is in line, behind the one being fetched now.',
    running: 'This video is being fetched now.',
    stopped: 'Fetching this video stopped part way.',
    done: 'A copy of this video is on this disk.',
    empty: 'There is no video at this address to copy.',
    failed: 'This video was not copied:',
  },
  article: {
    none: 'Nothing has been fetched from this address.',
    queued: 'This address is in line, behind the one being fetched now.',
    running: 'This address is being fetched now.',
    stopped: 'Fetching this address stopped part way.',
    done: 'What is at this address has already been fetched.',
    empty: 'This address publishes none of what was asked for.',
    failed: 'This address could not be reached:',
  },
}

/** What the list of vaults refused a command, in words a person reads. */
export const UNVAULTED: Record<VaultRefusalReason, string> = {
  unreadable: 'that folder is not there, or cannot be read',
  copy: 'that folder is a copy of a vault this installation already holds',
  overlaps: 'that folder is inside a vault already added, or holds one',
  nameTaken: 'a vault is already called that',
  lastVault: 'that is the only vault this installation has',
  showing: 'that is the vault in front of you',
  unknown: 'that vault is not on the list',
  noTrash: 'this machine has nowhere to put what is deleted',
  asking: 'a tab is holding text you have to answer for, so the window stayed where it was',
}

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
  ask: 'Ask the agent about this note',
  copy: 'Copy path',
  /** The two runs over the file in front: a recording transcribed, a scan recognised. */
  transcribe: 'Transcribe this recording',
  recognise: 'Recognise the text of this document',
  /** The transcript of the recording in front, put right by a proofreader. */
  proofread: 'Proofread the transcript of this recording',
  /** What is at the address a link note points at, fetched again. */
  fetch: 'Fetch what is at this address',
  /** A copy of the video a link note points at, fetched onto this disk. */
  download: 'Download a copy of this video',
  /** The transcript of the recording in front, taken away, and the two answers. */
  dropTranscript: 'Delete the transcript of this recording',
  keepsTranscript: 'Keep the transcript',
  drops: 'Delete',
  dropped: 'The words go, and the recording can be transcribed again',
  reveal: 'Show this note in the files',
  /** The preset this note is: the note itself, or the one a deck is scheduled by. */
  preset: 'Open the preset',
  /** The deck in front names no preset, so the defaults schedule it. */
  noPreset: 'This deck names no preset, so it is scheduled by the defaults.',
  newNote: note.newNote,
  newDeck: cards.newDeck,
  newStencil: cards.newStencil,
  /** The note that says how the decks pointing at it are scheduled. */
  newPreset: preset.made,
  /** The note that points at a web address. */
  importUrl: 'Import an address',
  newPlex: plex.newPlex,
  newAgent: agent.newAgent,
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
  /** Why nothing can be done to a note: the vault is unread, or none is in front. */
  indexing: 'The vault is still being read',
  noNote: 'Nothing in front of you is a note',
  /** The steps a command asks for: the chip beside the field, and the field. */
  command: 'Command',
  typeCommand: 'Type a command',
  naming: 'Name',
  typeName: 'What is it called',
  callIt: 'Call it',
  address: 'Address',
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
  /** What a command could not do, and what it left behind. */
  refused: REFUSED,
  unvaulted: UNVAULTED,
  dangling: 'These notes link to nothing now:',
  /** The vault opens with no note at all. */
  nowhere: 'The vault has no note to open with',
  unanswered: 'that note changed on disk, and its tab is waiting for an answer',
  overtaken: 'that note changed on disk while this was asked, so nothing was written',
  /** A file landed where something of its name is filed, and stayed where it was. */
  occupied: 'something of that name is filed there, so the file stayed where it was',
  /** This build cannot do the run at all, and stops offering it. */
  unrunnable: 'this installation of numen cannot do that at all',
  /** What an artifact of a file now stands at. */
  made: MADE,
  /** What the action panel of the palette is called. */
  actions: 'Actions',
  findAction: 'Search actions',
  noAction: 'Nothing by that name',
  /** The corner where what is running behind the window is shown. */
  wordsOnly: 'Searching by words only — no model set',
  working: 'Background work',
  putAway: 'Put away',
  more: 'more',
}
