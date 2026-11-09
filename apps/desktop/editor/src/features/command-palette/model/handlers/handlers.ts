/**
 * Carrying out a command over a note, a file or a tab.
 *
 * Every command is one entry in the table below, under the identity
 * `lib/table.ts` gives it, so a command that is offered and a command that
 * happens are the same list. What the window offers to do it with is
 * `types.ts`; what a command does to the vaults is `vaults.ts`. Nothing here
 * draws anything.
 */
import { formatErrorMessage } from '@numen/core-client'
import type { CommandDeps, CommandHandler } from './deps'
import type { CommandInvocation, CommandPlace } from '../../types'
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
import { ARGUMENT, COMMAND, SETTING_COMMAND_PREFIX } from '@/shared/commandIds'

const getPlace = (invocation: CommandInvocation): CommandPlace | undefined => {
  if (invocation.selector !== undefined) return { selector: invocation.selector }
  return invocation.line === undefined ? undefined : { line: invocation.line }
}

const openFile = (invocation: CommandInvocation, on: CommandDeps): void => {
  const file = invocation.path || invocation.file
  const showing = invocation.values?.[ARGUMENT.showing] === 'split' ? 'split' : 'current'
  on.notes.openFile(file, invocation.title || file, showing, getPlace(invocation))
}

/** What each command evaluates to. A command with no entry here does nothing. */
const HANDLERS: Record<string, CommandHandler> = {
  [COMMAND.openFile]: openFile,
  [COMMAND.createChildNote]: (invocation, on, words) =>
    createNoteCommand(invocation, on, { relation: 'child', words }),
  [COMMAND.createParentNote]: (invocation, on, words) =>
    createNoteCommand(invocation, on, { relation: 'parent', words }),
  [COMMAND.createJumpNote]: (invocation, on, words) =>
    createNoteCommand(invocation, on, { relation: 'jump', words }),
  [COMMAND.createNote]: (invocation, on, words) =>
    createNoteCommand(invocation, on, { relation: null, words }),
  [COMMAND.renameNote]: (invocation, on, words) => renameNoteCommand(invocation, on, words),
  [COMMAND.removeNote]: (invocation, on, words) =>
    removeFiles(invocation, on, {
      isPermanent: invocation.values?.[ARGUMENT.removal] === 'permanent',
      words,
    }),
  [COMMAND.copyNotePath]: (invocation, on) => on.copyPath(invocation.path),
  [COMMAND.openSettings]: (_, on) => on.navigator.openTab(SETTINGS),
  [COMMAND.moveFile]: (invocation, on, words) => moveFileCommand(invocation, on, words),
  [COMMAND.createFolder]: (invocation, on, words) => createFolderCommand(invocation, on, words),
  [COMMAND.closeTab]: (invocation, on) => on.navigator.closeTab(invocation.tab),
  [COMMAND.searchVault]: (_, on) => on.navigator.search(),
  [COMMAND.goToNote]: (invocation, on, words) => navigateToPath(invocation.path, on, words),
  [COMMAND.openVault]: (invocation, on, words) => showVault(invocation.vault.id, on, words),
  [COMMAND.createVault]: (_, on, words) => addVault(on, words),
  [COMMAND.renameVault]: (invocation, on, words) => renameVault(invocation, on, words),
  [COMMAND.removeVault]: (invocation, on, words) =>
    removeVault(invocation, on, { shouldErase: false, words }),
  [COMMAND.eraseVault]: (invocation, on, words) =>
    removeVault(invocation, on, { shouldErase: true, words }),
}

const commandRegistry = new Map<string, CommandHandler>()

export const commands = {
  registerCommand: (id: string, handler: CommandHandler): (() => void) => {
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

async function runSetting(invocation: CommandInvocation, on: CommandDeps): Promise<void> {
  try {
    await on.settings.chooseSetting(invocation.id, invocation.name)
  } catch (error) {
    on.writeMessage(formatErrorMessage(error), 'error')
  }
}

export async function runInvocation(
  invocation: CommandInvocation | null,
  on: CommandDeps,
  words: AnswerWords,
): Promise<void> {
  if (!invocation) return
  const isSetting = invocation.id.startsWith(SETTING_COMMAND_PREFIX)
  const handler = commandRegistry.get(invocation.id) ?? HANDLERS[invocation.id]
  if (!isSetting && !handler) return
  on.writeMessage('')
  on.emitEvent?.(`onCommand:${invocation.id}`)
  try {
    if (isSetting) {
      await runSetting(invocation, on)
    } else if (handler) {
      await handler(withCurrentNotePath(invocation, on), on, words)
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
  if (isWaiting) return on.writeMessage(words.unanswered, 'warning')
  on.writeMessage(formatNames(words.dangling, dangling), 'warning')
}

export * from './noteHandlers'
