/**
 * Carrying out a command over a note, a file or a tab.
 *
 * Every command is one entry in the table below, under the identity
 * `commands.ts` gives it, so a command that is offered and a command that
 * happens are the same list. What the window offers to do it with is
 * `deps.ts`; what a command does to the vaults is `vaults.ts`. Nothing here
 * draws anything.
 */
import { formatErrorMessage } from '@numen/wire'
import { all, naming, type CommandDeps, type CommandHandler, type Words } from './deps'
export type { CommandDeps }
import type { CommandInvocation } from './target'
import {
  atItsFile,
  createNoteCommand,
  createFolderCommand,
  moveFileCommand,
  navigateToPath,
  renameNoteCommand,
  settleTab,
} from './noteHandlers'
import { adds, calls, forgets, shows } from './vaults'
import type { Outcome, ArtifactState } from '../../shared/artifacts'
import { AGENT, FILES, PLEX, SETTINGS } from '../../entities/tab/workspace'

/** What each command comes to. A command with no entry here does nothing. */
const carried: Record<string, CommandHandler> = {
  read: (invocation, on) => on.notes.opens(invocation.path, invocation.title, 'here'),
  beside: (invocation, on) => on.notes.opens(invocation.path, invocation.title, 'beside'),
  travel: (invocation, on) => on.goes.travel(invocation.path),
  child: (invocation, on, words) => createNoteCommand(invocation, 'child', on, words),
  parent: (invocation, on, words) => createNoteCommand(invocation, 'parent', on, words),
  jump: (invocation, on, words) => createNoteCommand(invocation, 'jump', on, words),
  note: (invocation, on, words) => createNoteCommand(invocation, null, on, words),
  deck: async (invocation, on) => {
    if (invocation.name) await on.makers.decks('', invocation.name)
  },
  stencil: async (invocation, on) => {
    if (invocation.name) await on.makers.stencils('', invocation.name)
  },
  newPreset: async (invocation, on) => {
    if (invocation.name) await on.makers.presets('', invocation.name)
  },
  importUrl: async (invocation, on) => {
    if (invocation.name) await on.makers.imports('', invocation.name)
  },
  title: (invocation, on, words) => renameNoteCommand(invocation, on, words),
  remove: (invocation, on, words) => removes(invocation, false, on, words),
  destroy: (invocation, on, words) => removes(invocation, true, on, words),
  transcribe: async (invocation, on, words) =>
    began(invocation, await on.runs.createArtifact(invocation.file, 'transcript'), on, words),
  recognise: async (invocation, on, words) =>
    began(invocation, await on.runs.createArtifact(invocation.file, 'ocr'), on, words),
  downloadText: async (invocation, on, words) =>
    began(invocation, await on.runs.fetchArtifact(invocation.file), on, words),
  downloadCopy: async (invocation, on, words) =>
    began(invocation, await on.runs.createArtifact(invocation.file, 'copy'), on, words),
  proofread: async (invocation, on, words) =>
    began(invocation, await on.runs.correctArtifact(invocation.file), on, words),
  deleteText: async (invocation, on, words) => {
    if (await on.runs.deleteTranscript(invocation.file)) return
    on.runSupport.cannotRun(invocation.id)
    on.says(words.unrunnable, 'error')
  },
  deleteCopy: async (invocation, on, words) => {
    if (await on.runs.deleteCopy(invocation.file)) return
    on.runSupport.cannotRun(invocation.id)
    on.says(words.unrunnable, 'error')
  },
  ask: (invocation, on) => on.goes.asks(`${invocation.path} — `),
  copy: (invocation, on) => on.copies(invocation.path),
  reveal: (invocation, on) => on.goes.reveals(invocation.path),
  preset: (invocation, on) => on.goes.preset(invocation.path),
  settings: (_, on) => on.goes.opens(SETTINGS),
  move: (invocation, on, words) => moveFileCommand(invocation, on, words),
  makeFolder: (invocation, on, words) => createFolderCommand(invocation, on, words),
  plex: (_, on) => on.goes.opens(PLEX),
  files: (_, on) => on.goes.opens(FILES),
  agent: (_, on) => on.goes.opens(AGENT),
  close: (invocation, on) => on.goes.closes(invocation.tab),
  find: (_, on) => on.goes.searches(),
  appearance: (invocation, on) => on.settings.appearance(invocation.name),
  mode: (invocation, on) => on.settings.appearance(invocation.name),
  interfaceScale: (invocation, on) => on.settings.appearance(invocation.name),
  textScale: (invocation, on) => on.settings.appearance(invocation.name),
  syncing: (invocation, on) => on.settings.syncing(invocation.name),
  hanging: (invocation, on) => on.settings.hanging(invocation.name),
  parts: (invocation, on) => on.settings.parts(invocation.name),
  first: (_, on, words) => navigateToPath(on.goes.opening(), on, words),
  goto: (invocation, on, words) => navigateToPath(invocation.path, on, words),
  openVault: (invocation, on, words) => shows(invocation.vault.id, on, words),
  newVault: (_, on, words) => adds(on, words),
  renameVault: (invocation, on, words) => calls(invocation, on, words),
  forgetVault: (invocation, on, words) => forgets(invocation, false, on, words),
  eraseVault: (invocation, on, words) => forgets(invocation, true, on, words),
}

/** A command carried out. Nothing chosen does nothing at all. */
export async function does(invocation: CommandInvocation | null, on: CommandDeps, words: Words): Promise<void> {
  if (!invocation) return
  const carry = carried[invocation.id]
  if (!carry) return
  on.says('')
  try {
    await carry(atItsFile(invocation, on), on, words)
  } catch (error) {
    on.says(formatErrorMessage(error), 'error')
  }
}



/** What an artifact stands at when the ask did not come off, which is said as an error. */
const WENT_WRONG: readonly ArtifactState[] = ['none', 'stopped', 'empty', 'failed']

/**
 * An artifact asked for over a file. What it now stands at is one message,
 * which is what the person is told; a run under way shows what it is doing in
 * the work behind the window. A build that cannot make it at all is told once
 * and offers it nowhere after that.
 */
const began = (invocation: CommandInvocation, outcome: Outcome, on: CommandDeps, words: Words): void => {
  if (!outcome.able) {
    on.runSupport.cannotRun(invocation.id)
    return on.says(words.unrunnable, 'error')
  }
  // A file nothing here could read carries what the run said about it, and that
  // stands after the message.
  //
  // A run over an address is its own answer: it fetches whenever it is asked,
  // and what it fetched is not what a model heard in a recording.
  const why =
    invocation.id === 'downloadText' ? words.fetched[outcome.made] : words.made[outcome.of][outcome.made]
  const said = outcome.error ? `${why} ${outcome.error}` : why
  on.says(said, WENT_WRONG.includes(outcome.made) ? 'error' : 'report')
}


/**
 * The files one invocation is over: the one it names, and the rest of the selection it
 * was asked over.
 */
const over = (invocation: CommandInvocation): readonly string[] => [invocation.path, ...invocation.others]

/**
 * Files taken out of the vault. A file that has gone is gone from the tree, so
 * the notes left pointing at nothing are the whole of what is said. One the
 * vault refuses leaves the rest to go, and what was refused is what the person
 * is told.
 */
const removes = async (invocation: CommandInvocation, destroy: boolean, on: CommandDeps, words: Words): Promise<void> => {
  const dangling: string[] = []
  const refused: string[] = []
  let waiting = false
  const opening = on.goes.opening()

  for (const path of over(invocation)) {
    const tab = await settleTab(path, on)
    if (tab.waiting) {
      waiting = true
      continue
    }
    const answer = await on.files.removes(path, destroy)
    const error = answer.error
    if (error) {
      refused.push(words.errors[error])
      continue
    }
    if (tab.held) on.notes.shuts(tab.held)
    dangling.push(...answer.dangling)
    if (opening) await on.goes.leaves(path, opening)
  }

  if (refused.length > 0) return on.says(all(...refused), 'error')
  if (waiting) return on.says(words.unanswered, 'caution')
  on.says(naming(words.dangling, dangling))
}

export * from './noteHandlers'


