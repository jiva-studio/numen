/**
 * Carrying out a command of the window, and the registry the features and the plugins put their
 * commands in.
 *
 * A command is carried out by the handler registered under its id. The commands the palette
 * itself owns, the ones that close the tab in front, search the vault and open the settings, are in
 * the table below. What the window offers to do it with is `deps.ts`. Nothing here draws anything.
 */
import { formatError } from '@/shared/api/errors'
import {
  COMMAND,
  SETTING_COMMAND_PREFIX,
  type CommandDeps,
  type CommandHandler,
  type CommandInvocation,
} from '@/shared/model/command'
import { SETTINGS } from '@/entities/tab'

/** What each command of the palette evaluates to. A command with no entry here does nothing. */
const HANDLERS: Record<string, CommandHandler> = {
  [COMMAND.openSettings]: (_, on) => on.navigator.openTab(SETTINGS),
  [COMMAND.closeTab]: (invocation, on) => on.navigator.closeTab(invocation.target.tab),
  [COMMAND.searchVault]: (_, on) => on.navigator.search(),
}

async function runSetting(invocation: CommandInvocation, on: CommandDeps): Promise<void> {
  try {
    await on.settings.chooseSetting(invocation.id, invocation.name)
  } catch (error) {
    on.writeMessage(formatError(error), 'error')
  }
}

const withCurrentNotePath = (invocation: CommandInvocation, on: CommandDeps): CommandInvocation =>
  invocation.note
    ? { ...invocation, target: { ...invocation.target, path: on.notes.getPath(invocation.note) } }
    : invocation

export async function runInvocation(
  invocation: CommandInvocation | null,
  on: CommandDeps,
): Promise<string> {
  if (!invocation) return ''
  const isSetting = invocation.id.startsWith(SETTING_COMMAND_PREFIX)
  const handler = on.registry.get(invocation.id) ?? HANDLERS[invocation.id]
  if (!isSetting && !handler) return ''
  on.writeMessage('')
  on.emitEvent?.(`onCommand:${invocation.id}`)
  try {
    if (isSetting) {
      await runSetting(invocation, on)
    } else if (handler) {
      const made = await handler(withCurrentNotePath(invocation, on), on)
      return made || ''
    }
  } catch (error) {
    on.writeMessage(formatError(error), 'error')
  }
  return ''
}
