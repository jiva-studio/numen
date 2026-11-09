/**
 * The keys of the plugins as the window handles them: a key event is matched against the keybindings
 * of the registry, and the command of the winner is run over the note in front.
 */
import { isSharedCommand, type CommandNote, type CommandTarget } from '@numen/plugin-sdk'
import { APP_COMMAND_OWNER, getCommandOwner } from '../commandId'
import type { Keybinding, KeybindingRegistry } from '../keybindings/keybindingRegistry'
import type { WhenValues } from '../when/when'
import { getWhenValues } from '../when/whenValues'
import type { AvailableCommand } from './available'
import { reportError } from './events'
import type { CommandRun, HostDeps } from './types'

/** Runs a command of the application a key is bound to, with the values of the keybinding. */
export type RunAppKey = (command: string, values: Readonly<Record<string, string>>) => void

export interface ShortcutParts {
  readonly deps: HostDeps
  readonly keybindings: KeybindingRegistry
  readonly getAvailable: () => readonly AvailableCommand[]
  readonly executeCommand: (id: string, note: CommandNote, run?: CommandRun) => Promise<boolean>
}

/** The values the conditions of a key are judged over: those a slot is judged over, with the window's. */
function getContext(deps: HostDeps, target: CommandTarget | undefined): WhenValues {
  return (
    deps.getSlotWhenValues?.(target) ??
    getWhenValues({
      window: deps.getWindowValues?.(),
      target,
      contentTypes: deps.getContentTypes?.() ?? {},
    })
  )
}

/**
 * Creates the handler of key events. It returns whether the event is a key it runs, which the caller
 * answers by preventing the default of the event; that is known before the command runs. A key whose
 * command belongs to the application is handed to `runApp`, or left to the window where none is given;
 * a command of a plugin that has no handler yet is left to the window. With a target, the condition of
 * the command is judged over it as well as the key's own.
 */
export function createShortcutHandler(parts: ShortcutParts) {
  const { deps, keybindings } = parts
  return (
    event: KeyboardEvent,
    getNote: () => Promise<CommandNote>,
    target?: CommandTarget,
    runApp?: RunAppKey,
  ): boolean => {
    const context = getContext(deps, target)
    const available = new Map(parts.getAvailable().map((one) => [one.id, one]))
    const canRun = (binding: Keybinding): boolean => {
      if (getCommandOwner(binding.command) === APP_COMMAND_OWNER) return true
      if (isSharedCommand(binding.command)) {
        return (deps.getHandlerDeclarations?.() ?? []).some((one) => one.command === binding.command)
      }
      const command = available.get(binding.command)
      if (!command?.entry) return false
      if (target === undefined) return true
      return deps.whenReader.getCondition(command.declaration.when)?.evaluate(context) ?? false
    }
    const binding = keybindings.match(event, context, canRun)
    if (!binding) return false
    if (getCommandOwner(binding.command) === APP_COMMAND_OWNER) {
      if (!runApp) return false
      runApp(binding.command, binding.values)
      return true
    }
    const pluginId = available.get(binding.command)?.declaration.pluginId ?? binding.owner
    getNote()
      .then((note) => parts.executeCommand(binding.command, note, { values: binding.values }))
      .catch((error: unknown) => reportError(deps, pluginId, error))
    return true
  }
}
