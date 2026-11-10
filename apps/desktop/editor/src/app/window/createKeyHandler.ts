/**
 * The keys of the window: the tabs first, then the keybindings the application and the plugins
 * declare.
 */
import type { PluginHost } from '@numen/plugin-loader'
import { runInBackground } from '@numen/ui'
import type { Ref } from 'vue'
import {
  createInvocation,
  runInvocation,
  type createCommandPalette,
  type useSearch,
} from '@/features/command-palette'
import { syncKeybindings } from '@/features/hotkeys'
import {
  toRunTarget,
  type createCommandNoteReader,
  type DeclaredCommands,
} from '@/features/plugin-commands'
import { COMMAND, type CommandDeps, type CommandTarget } from '@/entities/command'
import type { WindowTabs } from '@/entities/tab'
import { formatError } from '@/shared/api/errors'
import type { MessageWriter } from '@/shared/model/notices/messages'
import { withValues, type Values } from './createCommandRunner'

interface KeyHandlerParts {
  readonly windowTabs: WindowTabs
  readonly pluginHost: PluginHost | undefined
  readonly declaredCommands: DeclaredCommands | undefined
  readonly plugins: Ref<readonly unknown[]>
  readonly palette: ReturnType<typeof useSearch>
  readonly commands: ReturnType<typeof createCommandPalette>
  readonly commandDeps: CommandDeps
  readonly runCommand: (id: string, target: CommandTarget) => void
  readonly getTarget: () => CommandTarget
  readonly getCommandNote: ReturnType<typeof createCommandNoteReader>
  readonly writeMessage: MessageWriter
}

export function createKeyHandler(parts: KeyHandlerParts): (event: KeyboardEvent) => void {
  const { windowTabs, pluginHost, palette, commands, commandDeps, getTarget, writeMessage } = parts
  const { runCommand, getCommandNote } = parts
  /**
   * Runs a command of the application over the front target with the values of its keybinding. The
   * command is not asked whether it is offered: the reveal of a note is offered only as the rows that
   * carry its values, and the condition of the key was judged when the key matched.
   */
  const runKeyWithValues = (command: string, values: Values): void => {
    const invocation = withValues(createInvocation(command, getTarget()), values)
    runInBackground(runInvocation(invocation, commandDeps), (error) =>
      writeMessage(formatError(error), 'error'),
    )
  }

  /** Runs the command of the application a key is bound to, with the values of the keybinding. */
  const runAppKey = (command: string, values: Values): void => {
    if (command === COMMAND.searchVault) {
      commands.palette.setOpen(false)
      palette.setOpen(!palette.open.value)
    } else if (command === COMMAND.openCommands) {
      palette.setOpen(false)
      commands.palette.setOpen(!commands.palette.open.value)
    } else if (Object.keys(values).length > 0) runKeyWithValues(command, values)
    else runCommand(command, getTarget())
  }

  if (pluginHost) {
    const keybindings = parts.declaredCommands?.contributions.keybindings
    if (keybindings) syncKeybindings(pluginHost, keybindings, parts.plugins)
  }

  /** The one handler of keys: the tabs first, then the registry of the keybindings. */
  return (event: KeyboardEvent) => {
    if (event.defaultPrevented) return
    if (windowTabs.panes.onKeyPress(event)) return event.preventDefault()
    const getNote = () => getCommandNote(getTarget())
    const keyContext = { target: toRunTarget(getTarget()), runApp: runAppKey }
    if (pluginHost?.handleShortcut(event, getNote, keyContext)) event.preventDefault()
  }
}
