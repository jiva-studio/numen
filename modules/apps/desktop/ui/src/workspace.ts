/**
 * What the window opens with, and what its tabs are called.
 *
 * The workspace takes identities and gives them back; which component each one
 * stands for is settled in `App.vue`.
 */
import { branch, pane, type WorkspaceLayout } from '@numen/ui'

/** The kinds of tab the window can open a second of. */
export const PLEX = 'plex'
export const AGENT = 'agent'
export const BLANK = 'blank'
export const NOTE = 'note'
/** A document read in the window, under the path the vault files it at. */
export const DOCUMENT = 'document'
/** The folders and files of the vault, one tab of them to a window. */
export const FILES = 'files'

/** One thread of talk, under the name the agent hears it by. */
export const CONVERSATION = 'conversation'

/**
 * A name no tab and no conversation has carried or will carry, in this window
 * or in one the person opens after it. The agent keeps a conversation under
 * each name it hears, and a name stands for one of them.
 */
export const named = (kind: string): string => `${kind}:${crypto.randomUUID()}`

/** What a plex tab is called: the note it stands on, under the word for a plex. */
export const plexCalled = (word: string, note: string): string => (note ? `${word} · ${note}` : word)

/**
 * What a tab is called by a question put in it: the first line with anything on
 * it, short enough to read at a glance. The cut lands at the last word to begin
 * past a third of the room, and mid-word when none does.
 */
export const shortened = (question: string, most = 24): string => {
  const line = question.split('\n').find((one) => one.trim() !== '')?.trim() ?? ''
  if (line.length <= most) return line
  const cut = line.slice(0, most)
  const space = cut.lastIndexOf(' ')
  return `${(space > most / 3 ? cut.slice(0, space) : cut).trimEnd()}…`
}

/**
 * The files along the leading edge, the plex with the room, and the agent along
 * the trailing edge. The person begins in the plex.
 */
export const opening = (files: string, plex: string, agent: string): WorkspaceLayout => ({
  root: branch(
    'root',
    [pane('files', [files]), pane('main', [plex]), pane('aside', [agent])],
    [0.18, 0.56, 0.26],
  ),
  axis: 'horizontal',
  focus: 'main',
})
