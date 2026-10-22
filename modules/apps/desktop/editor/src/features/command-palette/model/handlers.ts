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
import { joinAll, formatNames } from '../lib/voice'
export type { CommandDeps }
import {
  withCurrentNotePath,
  createNoteCommand,
  createFolderCommand,
  moveFileCommand,
  navigateToPath,
  renameNoteCommand,
  flushTab,
} from './noteHandlers'
import { addVault, renameVault, removeVault, showVault } from './vaults'
import type { Outcome, ArtifactState } from '@/entities/artifact'
import { AGENT, FILES, PLEX, SETTINGS } from '@/entities/tab'

/** What each command evaluates to. A command with no entry here does nothing. */
const HANDLERS: Record<string, CommandHandler> = {
  read: (invocation, on) => on.notes.openFile(invocation.path, invocation.title, 'here'),
  beside: (invocation, on) => on.notes.openFile(invocation.path, invocation.title, 'beside'),
  travel: (invocation, on) => on.navigator.travel(invocation.path),
  child: (invocation, on, words) => createNoteCommand(invocation, on, { relation: 'child', words }),
  parent: (invocation, on, words) =>
    createNoteCommand(invocation, on, { relation: 'parent', words }),
  jump: (invocation, on, words) => createNoteCommand(invocation, on, { relation: 'jump', words }),
  note: (invocation, on, words) => createNoteCommand(invocation, on, { relation: null, words }),
  title: (invocation, on, words) => renameNoteCommand(invocation, on, words),
  remove: (invocation, on, words) => removeFiles(invocation, on, { isPermanent: false, words }),
  destroy: (invocation, on, words) => removeFiles(invocation, on, { isPermanent: true, words }),
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
  ask: (invocation, on) => on.navigator.ask(`${invocation.path} — `),
  copy: (invocation, on) => on.copyPath(invocation.path),
  reveal: (invocation, on) => on.navigator.revealPath(invocation.path),
  preset: (invocation, on) => on.navigator.openPreset(invocation.path),
  settings: (_, on) => on.navigator.openTab(SETTINGS),
  move: (invocation, on, words) => moveFileCommand(invocation, on, words),
  createFolder: (invocation, on, words) => createFolderCommand(invocation, on, words),
  plex: (_, on) => on.navigator.openTab(PLEX),
  files: (_, on) => on.navigator.openTab(FILES),
  agent: (_, on) => on.navigator.openTab(AGENT),
  close: (invocation, on) => on.navigator.closeTab(invocation.tab),
  find: (_, on) => on.navigator.search(),
  appearance: (invocation, on) => on.settings.chooseAppearance(invocation.name),
  mode: (invocation, on) => on.settings.chooseAppearance(invocation.name),
  interfaceScale: (invocation, on) => on.settings.chooseAppearance(invocation.name),
  textScale: (invocation, on) => on.settings.chooseAppearance(invocation.name),
  syncing: (invocation, on) => on.settings.chooseSync(invocation.name),
  nesting: (invocation, on) => on.settings.chooseNesting(invocation.name),
  headings: (invocation, on) => on.settings.chooseHeadings(invocation.name),
  performanceProfile: (invocation, on) => on.settings.choosePerformance(invocation.name),
  indexingModel: (invocation, on) => on.settings.chooseIndexingModel(invocation.name),
  first: (_, on, words) => navigateToPath(on.navigator.getOpeningNote(), on, words),
  goto: (invocation, on, words) => navigateToPath(invocation.path, on, words),
  openVault: (invocation, on, words) => showVault(invocation.vault.id, on, words),
  newVault: (_, on, words) => addVault(on, words),
  renameVault: (invocation, on, words) => renameVault(invocation, on, words),
  removeVault: (invocation, on, words) =>
    removeVault(invocation, on, { shouldErase: false, words }),
  eraseVault: (invocation, on, words) => removeVault(invocation, on, { shouldErase: true, words }),
}

const commandRegistry = new Map<string, CommandHandler>()

export const commands = {
  registerCommand(id: string, handler: CommandHandler): () => void {
    commandRegistry.set(id, handler)
    return () => {
      if (commandRegistry.get(id) === handler) {
        commandRegistry.delete(id)
      }
    }
  },
  get(id: string): CommandHandler | undefined {
    return commandRegistry.get(id)
  },
  clear(): void {
    commandRegistry.clear()
  },
}

export const registerCommand = commands.registerCommand

type ArtifactAction = (runs: CommandDeps['runs'], file: string) => Promise<Outcome>

const DYNAMIC_ARTIFACTS: Record<string, ArtifactAction> = {
  transcribe: (runs, file) => runs.createArtifact(file, 'transcript'),
  recognise: (runs, file) => runs.createArtifact(file, 'ocr'),
  downloadText: (runs, file) => runs.fetchArtifact(file),
  downloadCopy: (runs, file) => runs.createArtifact(file, 'copy'),
  proofread: (runs, file) => runs.correctArtifact(file),
}

const DYNAMIC_CREATORS: Record<
  string,
  (makers: CommandDeps['makers'], name: string) => Promise<unknown>
> = {
  deck: (makers, name) =>
    'createAndOpenFile' in makers &&
    typeof (makers as Record<string, unknown>).createAndOpenFile === 'function'
      ? (
          makers as {
            createAndOpenFile: (k: string, f: string, n: string) => Promise<unknown>
          }
        ).createAndOpenFile('deck', '', name)
      : makers.createDeck('', name),
  stencil: (makers, name) =>
    'createAndOpenFile' in makers &&
    typeof (makers as Record<string, unknown>).createAndOpenFile === 'function'
      ? (
          makers as {
            createAndOpenFile: (k: string, f: string, n: string) => Promise<unknown>
          }
        ).createAndOpenFile('stencil', '', name)
      : makers.createStencil('', name),
  newPreset: (makers, name) =>
    'createAndOpenFile' in makers &&
    typeof (makers as Record<string, unknown>).createAndOpenFile === 'function'
      ? (
          makers as {
            createAndOpenFile: (k: string, f: string, n: string) => Promise<unknown>
          }
        ).createAndOpenFile('preset', '', name)
      : makers.createPreset('', name),
  importUrl: (makers, name) =>
    'createAndOpenFile' in makers &&
    typeof (makers as Record<string, unknown>).createAndOpenFile === 'function'
      ? (
          makers as {
            createAndOpenFile: (k: string, f: string, n: string) => Promise<unknown>
          }
        ).createAndOpenFile('url', '', name)
      : makers.createUrl('', name),
}

function resolveDynamicHandler(id: string): CommandHandler | undefined {
  const artifactOp = DYNAMIC_ARTIFACTS[id]
  if (artifactOp) {
    return async (invocation, on, words) => {
      const outcome = await artifactOp(on.runs, invocation.file)
      reportOutcome(invocation, outcome, { on, words })
    }
  }

  const creatorOp = DYNAMIC_CREATORS[id]
  if (creatorOp) {
    return async (invocation, on) => {
      if (invocation.name) {
        await creatorOp(on.makers, invocation.name)
      }
    }
  }

  return undefined
}

async function handlePluginInvocation(
  invocation: CommandInvocation,
  on: CommandDeps,
): Promise<void> {
  const [, pluginId, key] = invocation.id.split(':')
  if (!pluginId || !key || !on.settings.choosePluginSetting) return
  let value: unknown = invocation.name
  if (invocation.name === 'on') {
    value = true
  } else if (invocation.name === 'off') {
    value = false
  }
  await on.settings.choosePluginSetting(pluginId, key, value)
}

export async function runInvocation(
  invocation: CommandInvocation | null,
  on: CommandDeps,
  words: AnswerWords,
): Promise<void> {
  if (!invocation) return
  const isPlugin = invocation.id.startsWith('plugin-setting:')
  const handler =
    commandRegistry.get(invocation.id) ??
    HANDLERS[invocation.id] ??
    resolveDynamicHandler(invocation.id)
  if (!isPlugin && !handler) return
  on.writeMessage('')
  on.emitEvent?.(`onCommand:${invocation.id}`)
  try {
    if (isPlugin) {
      await handlePluginInvocation(invocation, on)
    } else {
      await handler(withCurrentNotePath(invocation, on), on, words)
    }
  } catch (error) {
    on.writeMessage(formatErrorMessage(error), 'error')
  }
}

const ERROR_STATES: readonly ArtifactState[] = ['none', 'stopped', 'empty', 'failed']

/**
 * the work behind the window. A build that cannot make it at all is told once
 * and offers it nowhere after that.
 */
interface ReportOutcomeDeps {
  readonly on: RunContext & Voice
  readonly words: AnswerWords
}

const reportOutcome = (
  invocation: CommandInvocation,
  outcome: Outcome,
  deps: ReportOutcomeDeps,
): void => {
  const { on, words } = deps
  if (!outcome.able) {
    on.runSupport.cannotRun(invocation.id)
    return on.writeMessage(words.unrunnable, 'error')
  }
  const why =
    invocation.id === 'downloadText'
      ? words.fetched[outcome.made]
      : words.made[outcome.of][outcome.made]
  const message = outcome.error ? `${why} ${outcome.error}` : why
  on.writeMessage(message, ERROR_STATES.includes(outcome.made) ? 'error' : 'report')
}

const getInvocationPaths = (invocation: CommandInvocation): readonly string[] => [
  invocation.path,
  ...invocation.others,
]

interface RemoveFilesOptions {
  readonly isPermanent: boolean
  readonly words: AnswerWords
}

const removeFiles = async (
  invocation: CommandInvocation,
  on: CommandDeps,
  options: RemoveFilesOptions,
): Promise<void> => {
  const { isPermanent, words } = options
  const dangling: string[] = []
  const errors: string[] = []
  let isWaiting = false
  const opening = on.navigator.getOpeningNote()

  for (const path of getInvocationPaths(invocation)) {
    const tab = await flushTab(path, on)
    if (tab.isWaiting) {
      isWaiting = true
      continue
    }
    const answer = await on.files.remove(path, isPermanent)
    if (!answer.ok) {
      errors.push(words.errors[answer.error])
      continue
    }
    if (tab.tabId) on.notes.close(tab.tabId)
    dangling.push(...answer.value.dangling)
    if (opening) await on.navigator.leave(path, opening)
  }

  if (errors.length > 0) return on.writeMessage(joinAll(...errors), 'error')
  if (isWaiting) return on.writeMessage(words.unanswered, 'caution')
  on.writeMessage(formatNames(words.dangling, dangling))
}

export * from './noteHandlers'
