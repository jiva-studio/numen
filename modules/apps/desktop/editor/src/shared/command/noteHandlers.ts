/**
 * Note and file creation and mutation helpers for command execution.
 */
import type { PlexRelatedSeat } from '@numen/ui'
import type { CommandDeps, Words } from './deps'
import type { CommandInvocation } from './target'
import { NOTE } from '../tabs/workspace'

/** A tab asked to settle: which one it was, and whether it is still waiting. */
export interface SettleResult {
  readonly held: string | null
  readonly waiting: boolean
}

/**
 * The invocation at the file its note stands at now. One over a note no tab of the
 * window holds is at the name it was made over.
 */
export const atItsFile = (invocation: CommandInvocation, on: CommandDeps): CommandInvocation =>
  invocation.note ? { ...invocation, path: on.notes.where(invocation.note) } : invocation

/**
 * The tab holding a note, once nothing of the note is on its way to its file.
 * A tab waiting on the person to answer for it settles nothing and says so.
 */
export const settles = async (path: string, on: CommandDeps): Promise<SettleResult> => {
  const held = on.notes.holding(path)
  if (held === null) return { held, waiting: false }
  if (on.notes.asking(held)) return { held, waiting: true }
  await on.notes.settles(held)
  return { held, waiting: false }
}

/** A note travelled to, and a vault with none to travel to said. */
export const travels = async (path: string, on: CommandDeps, words: Words): Promise<void> => {
  if (!path) return on.says(words.nowhere, 'caution')
  await on.goes.travel(path)
}

/**
 * A note made under the name that was typed. From a note tab it opens in a tab
 * beside; anywhere else the plex the person is looking at travels to it.
 */
export const makes = async (
  invocation: CommandInvocation,
  seat: PlexRelatedSeat | null,
  on: CommandDeps,
  words: Words,
): Promise<void> => {
  if (!invocation.name) return
  const made = await on.files.makes(invocation.name, seat ? invocation.path : '', seat)
  if (!made) return
  if (invocation.kind === NOTE) return on.notes.made(made.path, made.title, 'note', 'beside')
  await travels(made.path, on, words)
}

/**
 * A note given a different name, and its file renamed with it where the two are
 * one name. Prose on disk that nobody here has seen leaves the note as it is.
 */
export const renames = async (invocation: CommandInvocation, on: CommandDeps, words: Words): Promise<void> => {
  if (!invocation.name || invocation.name === invocation.title) return
  const tab = await settles(invocation.path, on)
  if (tab.waiting) return on.says(words.unanswered, 'caution')
  const answer = await on.files.renames(invocation.path, invocation.name)
  if (answer.changed) return on.says(words.stale ?? words.overtaken ?? '', 'caution')
  const error = answer.error ?? answer.refusal
  if (error) on.says((words.errors ?? words.refused)[error], 'refusal')
}

/**
 * A file or a folder filed somewhere else, carrying the name the path ends in.
 * A destination that is taken leaves it where it was.
 */
export const moves = async (invocation: CommandInvocation, on: CommandDeps, words: Words): Promise<void> => {
  if (!invocation.name || invocation.name === invocation.path) return
  const tab = await settles(invocation.path, on)
  if (tab.waiting) return on.says(words.unanswered, 'caution')
  const answer = await on.files.moves(invocation.path, invocation.name)
  const error = answer.error ?? answer.refusal
  if (error === 'occupied') return on.says(words.occupied, 'refusal')
  if (error) on.says((words.errors ?? words.refused)[error], 'refusal')
}

/** An empty folder, made under the path that was typed. */
export const makesFolder = async (invocation: CommandInvocation, on: CommandDeps, words: Words): Promise<void> => {
  if (!invocation.name) return
  const refusal = await on.files.makesFolder(invocation.name)
  if (refusal === 'occupied') return on.says(words.occupied, 'refusal')
  if (refusal) on.says(words.refused[refusal], 'refusal')
}
