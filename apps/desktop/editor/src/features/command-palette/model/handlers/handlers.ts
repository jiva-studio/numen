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
import type { CommandDeps, CommandHandler } from './deps'
import type { CommandInvocation } from '../../types'
import type { AnswerWords } from '../../words'
import { joinAll, formatNames } from '../../lib/voice/voice'
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
import { SETTINGS } from '@/entities/tab'

/** What each command evaluates to. A command with no entry here does nothing. */
const HANDLERS: Record<string, CommandHandler> = {
  read: (invocation, on) => on.notes.openFile(invocation.path, invocation.title, 'here'),
  beside: (invocation, on) => on.notes.openFile(invocation.path, invocation.title, 'beside'),
  child: (invocation, on, words) => createNoteCommand(invocation, on, { relation: 'child', words }),
  parent: (invocation, on, words) =>
    createNoteCommand(invocation, on, { relation: 'parent', words }),
  jump: (invocation, on, words) => createNoteCommand(invocation, on, { relation: 'jump', words }),
  note: (invocation, on, words) => createNoteCommand(invocation, on, { relation: null, words }),
  title: (invocation, on, words) => renameNoteCommand(invocation, on, words),
  remove: (invocation, on, words) => removeFiles(invocation, on, { isPermanent: false, words }),
  destroy: (invocation, on, words) => removeFiles(invocation, on, { isPermanent: true, words }),
  copy: (invocation, on) => on.copyPath(invocation.path),
  settings: (_, on) => on.navigator.openTab(SETTINGS),
  move: (invocation, on, words) => moveFileCommand(invocation, on, words),
  createFolder: (invocation, on, words) => createFolderCommand(invocation, on, words),
  close: (invocation, on) => on.navigator.closeTab(invocation.tab),
  find: (_, on) => on.navigator.search(),
  appearance: (invocation, on) => on.settings.chooseAppearance(invocation.name),
  mode: (invocation, on) => on.settings.chooseAppearance(invocation.name),
  interfaceScale: (invocation, on) => on.settings.chooseAppearance(invocation.name),
  textScale: (invocation, on) => on.settings.chooseAppearance(invocation.name),
  syncing: (invocation, on) => on.settings.chooseSync(invocation.name),
  performanceProfile: (invocation, on) => on.settings.choosePerformance(invocation.name),
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
  const handler = commandRegistry.get(invocation.id) ?? HANDLERS[invocation.id]
  if (!isPlugin && !handler) return
  on.writeMessage('')
  on.emitEvent?.(`onCommand:${invocation.id}`)
  try {
    if (handler && !isPlugin) {
      await handler(withCurrentNotePath(invocation, on), on, words)
    } else {
      await handlePluginInvocation(invocation, on)
    }
  } catch (error) {
    on.writeMessage(formatErrorMessage(error), 'error')
  }
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
