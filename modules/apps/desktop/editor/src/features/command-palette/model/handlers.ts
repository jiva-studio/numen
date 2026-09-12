/**
 * Carrying out a command over a note, a file or a tab.
 *
 * Every command is one entry in the table below, under the identity
 * `lib/table.ts` gives it, so a command that is offered and a command that
 * happens are the same list. What the window offers to do it with is
 * `types.ts`; what a command does to the vaults is `vaults.ts`. Nothing here
 * draws anything.
 */
import { formatErrorMessage } from '@numen/wire'
import type { CommandDeps, CommandHandler, RunContext, Voice } from './deps'
import type { CommandInvocation } from '../types'
import type { AnswerWords } from '../words'
import { all, formatNames } from '../lib/voice'
export type { CommandDeps }
import {
  atItsFile,
  createNoteCommand,
  createFolderCommand,
  moveFileCommand,
  navigateToPath,
  renameNoteCommand,
  settleTab,
} from './noteHandlers'
import { addVault, renameVault, removeVault, showVault } from './vaults'
import type { Outcome, ArtifactState } from '@/shared/artifacts'
import { AGENT, FILES, PLEX, SETTINGS } from '@/entities/tab'

/** What each command comes to. A command with no entry here does nothing. */
const HANDLERS: Record<string, CommandHandler> = {
  read: (invocation, on) => on.notes.openFile(invocation.path, invocation.title, 'here'),
  beside: (invocation, on) => on.notes.openFile(invocation.path, invocation.title, 'beside'),
  travel: (invocation, on) => on.goes.travel(invocation.path),
  child: (invocation, on, words) => createNoteCommand(invocation, 'child', on, words),
  parent: (invocation, on, words) => createNoteCommand(invocation, 'parent', on, words),
  jump: (invocation, on, words) => createNoteCommand(invocation, 'jump', on, words),
  note: (invocation, on, words) => createNoteCommand(invocation, null, on, words),
  deck: async (invocation, on) => {
    if (invocation.name) await on.makers.createDeck('', invocation.name)
  },
  stencil: async (invocation, on) => {
    if (invocation.name) await on.makers.createStencil('', invocation.name)
  },
  newPreset: async (invocation, on) => {
    if (invocation.name) await on.makers.createPreset('', invocation.name)
  },
  importUrl: async (invocation, on) => {
    if (invocation.name) await on.makers.createUrl('', invocation.name)
  },
  title: (invocation, on, words) => renameNoteCommand(invocation, on, words),
  remove: (invocation, on, words) => removeFiles(invocation, false, on, words),
  destroy: (invocation, on, words) => removeFiles(invocation, true, on, words),
  transcribe: async (invocation, on, words) =>
    reportOutcome(invocation, await on.runs.createArtifact(invocation.file, 'transcript'), on, words),
  recognise: async (invocation, on, words) =>
    reportOutcome(invocation, await on.runs.createArtifact(invocation.file, 'ocr'), on, words),
  downloadText: async (invocation, on, words) =>
    reportOutcome(invocation, await on.runs.fetchArtifact(invocation.file), on, words),
  downloadCopy: async (invocation, on, words) =>
    reportOutcome(invocation, await on.runs.createArtifact(invocation.file, 'copy'), on, words),
  proofread: async (invocation, on, words) =>
    reportOutcome(invocation, await on.runs.correctArtifact(invocation.file), on, words),
  deleteText: async (invocation, on, words) => {
    if (await on.runs.deleteTranscript(invocation.file)) return
    on.runSupport.cannotRun(invocation.id)
    on.writeMessage(words.unrunnable, 'error')
  },
  deleteCopy: async (invocation, on, words) => {
    if (await on.runs.deleteCopy(invocation.file)) return
    on.runSupport.cannotRun(invocation.id)
    on.writeMessage(words.unrunnable, 'error')
  },
  ask: (invocation, on) => on.goes.ask(`${invocation.path} — `),
  copy: (invocation, on) => on.copyPath(invocation.path),
  reveal: (invocation, on) => on.goes.revealPath(invocation.path),
  preset: (invocation, on) => on.goes.openPreset(invocation.path),
  settings: (_, on) => on.goes.openTab(SETTINGS),
  move: (invocation, on, words) => moveFileCommand(invocation, on, words),
  createFolder: (invocation, on, words) => createFolderCommand(invocation, on, words),
  plex: (_, on) => on.goes.openTab(PLEX),
  files: (_, on) => on.goes.openTab(FILES),
  agent: (_, on) => on.goes.openTab(AGENT),
  close: (invocation, on) => on.goes.closeTab(invocation.tab),
  find: (_, on) => on.goes.search(),
  appearance: (invocation, on) => on.settings.chooseAppearance(invocation.name),
  mode: (invocation, on) => on.settings.chooseAppearance(invocation.name),
  interfaceScale: (invocation, on) => on.settings.chooseAppearance(invocation.name),
  textScale: (invocation, on) => on.settings.chooseAppearance(invocation.name),
  syncing: (invocation, on) => on.settings.chooseSync(invocation.name),
  hanging: (invocation, on) => on.settings.chooseHanging(invocation.name),
  parts: (invocation, on) => on.settings.chooseParts(invocation.name),
  first: (_, on, words) => navigateToPath(on.goes.getOpeningNote(), on, words),
  goto: (invocation, on, words) => navigateToPath(invocation.path, on, words),
  openVault: (invocation, on, words) => showVault(invocation.vault.id, on, words),
  newVault: (_, on, words) => addVault(on, words),
  renameVault: (invocation, on, words) => renameVault(invocation, on, words),
  forgetVault: (invocation, on, words) => removeVault(invocation, false, on, words),
  eraseVault: (invocation, on, words) => removeVault(invocation, true, on, words),
}

/** A command carried out. Nothing chosen does nothing at all. */
export async function runInvocation(invocation: CommandInvocation | null, on: CommandDeps, words: AnswerWords): Promise<void> {
  if (!invocation) return
  const handler = HANDLERS[invocation.id]
  if (!handler) return
  on.writeMessage('')
  try {
    await handler(atItsFile(invocation, on), on, words)
  } catch (error) {
    on.writeMessage(formatErrorMessage(error), 'error')
  }
}



/** What an artifact stands at when the ask did not come off, which is said as an error. */
const ERROR_STATES: readonly ArtifactState[] = ['none', 'stopped', 'empty', 'failed']

/**
 * An artifact asked for over a file. What it now stands at is one message,
 * which is what the person is told; a run under way shows what it is doing in
 * the work behind the window. A build that cannot make it at all is told once
 * and offers it nowhere after that.
 */
const reportOutcome = (invocation: CommandInvocation, outcome: Outcome, on: RunContext & Voice, words: AnswerWords): void => {
  if (!outcome.able) {
    on.runSupport.cannotRun(invocation.id)
    return on.writeMessage(words.unrunnable, 'error')
  }
  // A file nothing here could read carries what the run said about it, and that
  // stands after the message.
  //
  // A run over an address is its own answer: it fetches whenever it is asked,
  // and what it fetched is not what a model heard in a recording.
  const why =
    invocation.id === 'downloadText' ? words.fetched[outcome.made] : words.made[outcome.of][outcome.made]
  const said = outcome.error ? `${why} ${outcome.error}` : why
  on.writeMessage(said, ERROR_STATES.includes(outcome.made) ? 'error' : 'report')
}


/**
 * The files one invocation is over: the one it names, and the rest of the selection it
 * was asked over.
 */
const getInvocationPaths = (invocation: CommandInvocation): readonly string[] => [invocation.path, ...invocation.others]

/**
 * Files taken out of the vault. A file that has gone is gone from the tree, so
 * the notes left pointing at nothing are the whole of what is said. One the
 * vault refuses leaves the rest to go, and what was refused is what the person
 * is told.
 */
const removeFiles = async (
  invocation: CommandInvocation,
  destroy: boolean,
  on: CommandDeps,
  words: AnswerWords,
): Promise<void> => {
  const dangling: string[] = []
  const errors: string[] = []
  let waiting = false
  const opening = on.goes.getOpeningNote()

  for (const path of getInvocationPaths(invocation)) {
    const tab = await settleTab(path, on)
    if (tab.waiting) {
      waiting = true
      continue
    }
    const answer = await on.files.remove(path, destroy)
    const error = answer.error
    if (error) {
      errors.push(words.errors[error])
      continue
    }
    if (tab.held) on.notes.close(tab.held)
    dangling.push(...answer.dangling)
    if (opening) await on.goes.leave(path, opening)
  }

  if (errors.length > 0) return on.writeMessage(all(...errors), 'error')
  if (waiting) return on.writeMessage(words.unanswered, 'caution')
  on.writeMessage(formatNames(words.dangling, dangling))
}

export * from './noteHandlers'


