/**
 * The keys of the window: the tabs first, then the keybindings the application and the plugins
 * declare.
 */
import type { Ref } from 'vue'
import type { PluginHost } from '@numen/plugin-loader'
import { COMMAND, type CommandTarget, type Values } from '@/shared/model/command'
import type { WindowTabs } from '@/entities/tab'
import type { DeclaredKeybinding } from '@/entities/plugin'
import { syncKeybindings } from './syncKeybindings'

/** What the keys of the window reach. */
export interface KeyHandlerParts {
  readonly windowTabs: { readonly panes: Pick<WindowTabs['panes'], 'onKeyPress'> }
  readonly keybindings: PluginHost['keybindings']
  readonly declaredKeybindings: Ref<readonly DeclaredKeybinding[] | undefined>
  readonly plugins: Ref<readonly object[]>
  readonly handleShortcut: (
    event: KeyboardEvent,
    runApp: (command: string, values: Values) => void,
  ) => boolean
  readonly toggleSearch: () => void
  readonly toggleCommands: () => void
  readonly runCommand: (id: string, target: CommandTarget) => void
  readonly runWithValues: (id: string, target: CommandTarget, values: Values) => void
  readonly getTarget: () => CommandTarget
}

export function createKeyHandler(parts: KeyHandlerParts): (event: KeyboardEvent) => void {
  const { windowTabs, getTarget } = parts

  /** Runs the command of the application a key is bound to, with the values of the keybinding. */
  const runAppKey = (command: string, values: Values): void => {
    if (command === COMMAND.searchVault) parts.toggleSearch()
    else if (command === COMMAND.openCommands) parts.toggleCommands()
    else if (Object.keys(values).length > 0) parts.runWithValues(command, getTarget(), values)
    else parts.runCommand(command, getTarget())
  }

  syncKeybindings(parts.keybindings, parts.declaredKeybindings, parts.plugins)

  /** The one handler of keys: the tabs first, then the registry of the keybindings. */
  return (event: KeyboardEvent) => {
    if (event.defaultPrevented) return
    if (windowTabs.panes.onKeyPress(event)) return event.preventDefault()
    if (parts.handleShortcut(event, runAppKey)) event.preventDefault()
  }
}
