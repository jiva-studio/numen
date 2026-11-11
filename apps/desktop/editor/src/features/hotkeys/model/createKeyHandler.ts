/**
 * The keys of the window: the tabs first, then the keybindings the application and the plugins
 * declare.
 */
import { COMMAND, type CommandTarget, type Values } from '@/shared/model/command'
import type { WindowTabs } from '@/entities/tab'

/** What a key bound to a command of the application reaches. */
export interface AppKeyParts {
  readonly toggleSearch: () => void
  readonly toggleCommands: () => void
  readonly runCommand: (id: string, target: CommandTarget) => void
  readonly runWithValues: (id: string, target: CommandTarget, values: Values) => void
  readonly getTarget: () => CommandTarget
}

/** What the keys of the window reach. */
export interface KeyHandlerParts {
  readonly windowTabs: { readonly panes: Pick<WindowTabs['panes'], 'onKeyPress'> }
  readonly handleShortcut: (
    event: KeyboardEvent,
    runApp: (command: string, values: Values) => void,
  ) => boolean
  readonly runApp: (command: string, values: Values) => void
}

/** Runs the command of the application a key is bound to, with the values of the keybinding. */
export function createAppKeyRunner(parts: AppKeyParts): (command: string, values: Values) => void {
  return (command, values) => {
    if (command === COMMAND.searchVault) parts.toggleSearch()
    else if (command === COMMAND.openCommands) parts.toggleCommands()
    else if (Object.keys(values).length > 0) parts.runWithValues(command, parts.getTarget(), values)
    else parts.runCommand(command, parts.getTarget())
  }
}

export function createKeyHandler(parts: KeyHandlerParts): (event: KeyboardEvent) => void {
  const { windowTabs } = parts

  /** The one handler of keys: the tabs first, then the registry of the keybindings. */
  return (event: KeyboardEvent) => {
    if (event.defaultPrevented) return
    if (windowTabs.panes.onKeyPress(event)) return event.preventDefault()
    if (parts.handleShortcut(event, parts.runApp)) event.preventDefault()
  }
}
