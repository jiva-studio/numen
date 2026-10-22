/**
 * Note and file creation and mutation helpers for command execution.
 */
import type { PlexRelation } from '@numen/ui'
import type { CommandDeps, TabContext, VaultContext, Voice } from './deps'
import type { CommandInvocation } from '../../types'
import type { AnswerWords } from '../../words'
import { NOTE } from '@/entities/tab'

/** A tab asked to flush its pending write: which one it was, and whether it is still waiting. */
export interface FlushResult {
  readonly tabId: string | null
  readonly isWaiting: boolean
}

/**
 * window holds is at the name it was made over.
 */
export const withCurrentNotePath = (
  invocation: CommandInvocation,
  on: TabContext,
): CommandInvocation =>
  invocation.note ? { ...invocation, path: on.notes.getPath(invocation.note) } : invocation

/**
 * The tab holding a note, once nothing of the note is on its way to its file.
 * A tab waiting on the person to answer for it flushes nothing and says so.
 */
export const flushTab = async (path: string, on: TabContext): Promise<FlushResult> => {
  const tabId = on.notes.getTabAt(path)
  if (tabId === null) return { tabId, isWaiting: false }
  if (on.notes.asking(tabId)) return { tabId, isWaiting: true }
  await on.notes.flushPending(tabId)
  return { tabId, isWaiting: false }
}

/** A note travelled to, and a vault with none to travel to said. */
export const navigateToPath = async (
  path: string,
  on: TabContext & Voice,
  words: AnswerWords,
): Promise<void> => {
  if (!path) return on.writeMessage(words.nowhere, 'caution')
  await on.navigator.travel(path)
}

export interface CreateNoteOptions {
  readonly relation: PlexRelation | null
  readonly words: AnswerWords
}

export const createNoteCommand = async (
  invocation: CommandInvocation,
  on: CommandDeps,
  options: CreateNoteOptions,
): Promise<void> => {
  if (!invocation.name) return
  const { relation, words } = options
  const createdItem = await on.files.createNote(
    invocation.name,
    relation ? invocation.path : '',
    relation,
  )
  if (!createdItem) return
  if (invocation.kind === NOTE) {
    return on.notes.openNewFile(createdItem.path, createdItem.title, {
      type: 'note',
      where: 'beside',
    })
  }
  await navigateToPath(createdItem.path, on, words)
}

/**
 * A note given a different name, and its file renamed with it where the two are
 * one name. Prose on disk that nobody here has seen leaves the note as it is.
 */
export const renameNoteCommand = async (
  invocation: CommandInvocation,
  on: CommandDeps,
  words: AnswerWords,
): Promise<void> => {
  if (!invocation.name || invocation.name === invocation.title) return
  const tab = await flushTab(invocation.path, on)
  if (tab.isWaiting) return on.writeMessage(words.unanswered, 'caution')
  const answer = await on.files.rename(invocation.path, invocation.name)
  if (answer.ok) return
  if (answer.error === 'changed') return on.writeMessage(words.stale, 'caution')
  on.writeMessage(words.errors[answer.error], 'error')
}

/**
 * A file or a folder filed somewhere else, carrying the name the path ends in.
 * A destination that is taken leaves it where it was.
 */
export const moveFileCommand = async (
  invocation: CommandInvocation,
  on: CommandDeps,
  words: AnswerWords,
): Promise<void> => {
  if (!invocation.name || invocation.name === invocation.path) return
  const tab = await flushTab(invocation.path, on)
  if (tab.isWaiting) return on.writeMessage(words.unanswered, 'caution')
  const answer = await on.files.move(invocation.path, invocation.name)
  if (answer.ok) return
  if (answer.error === 'occupied') return on.writeMessage(words.occupied, 'error')
  on.writeMessage(words.errors[answer.error], 'error')
}

/** An empty folder, made under the path that was typed. */
export const createFolderCommand = async (
  invocation: CommandInvocation,
  on: VaultContext & Voice,
  words: AnswerWords,
): Promise<void> => {
  if (!invocation.name) return
  const error = await on.files.createFolder(invocation.name)
  if (error === 'occupied') return on.writeMessage(words.occupied, 'error')
  if (error) on.writeMessage(words.errors[error], 'error')
}
