/**
 * What the window opens with, and what its tabs are called.
 *
 * The workspace takes identities and gives them back; which component each one
 * stands for is settled in `App.vue`.
 */
import { branch, pane, type Workspace } from '@numen/ui'

/** The kinds of tab the window can open a second of. */
export const PLEX = 'plex'
export const AGENT = 'agent'
export const NOTE = 'note'
/** A document read in the window, under the path the vault files it at. */
export const DOCUMENT = 'document'
/** A book that reflows, read in the window, under the path the vault files it at. */
export const BOOK = 'book'
/** A recording played in the window, under the path the vault files it at. */
export const RECORDING = 'recording'
/** A url opened in the window, under the path the vault files it at. */
export const URL = 'url'
/** The folders and files of the vault, one tab of them to a window. */
export const FILES = 'files'
/** A deck edited in the window, under the path the vault files it at. */
export const DECK = 'deck'
/** A stencil edited in the window, under the path the vault files it at. */
export const STENCIL = 'stencil'
/** A preset edited in the window, under the path the vault files it at. */
export const PRESET = 'preset'
/** What this installation is configured as, one tab of it to a window. */
export const SETTINGS = 'settings'
/** The settings file itself, opened whole as text, one tab of it to a window. */
export const SETTINGS_FILE = 'settings-file'

/** One thread of talk, under the name the agent hears it by. */
export const CONVERSATION = 'conversation'

/**
 * The identity a tab or a conversation is given when nothing it can be found
 * under exists: the word its kind is filed under, and what nothing else
 * answers to, in this window or in one the person opens after it. The agent
 * keeps a conversation under each name it hears, and a name stands for one of
 * them.
 */
export const generateId = (kind: string): string => `${kind}:${crypto.randomUUID()}`

/**
 * The plex with the room, and one pane along the trailing edge holding the
 * agent in front of the files. The person begins in the plex.
 */
export const begun = (plex: string, agent: string, files: string): Workspace => ({
  root: branch('root', [pane('main', [plex]), pane('aside', [agent, files])], [0.72, 0.28]),
  axis: 'horizontal',
  focus: 'main',
})
