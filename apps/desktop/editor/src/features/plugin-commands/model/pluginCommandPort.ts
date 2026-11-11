/**
 * The commands the plugins declare, as the window runs them: from the palette, from a key, and to
 * their end for a plugin that awaits one.
 */
import type { PluginHost } from '@numen/plugin-loader'
import type { CommandTarget, Values } from '@/entities/command'
import { toRunTarget } from '../lib/runTarget'
import type { createCommandNoteReader } from './usePluginCommands'

/** How a command of a plugin is run: with the values it is given, and whether the caller awaits it. */
interface PluginRun {
  readonly values?: Values
  readonly isAwaited?: true
}

export function createPluginCommandPort(
  pluginHost: Pick<PluginHost, 'commands'>,
  getCommandNote: ReturnType<typeof createCommandNoteReader>,
) {
  const { commands } = pluginHost
  return {
    /** Whether the command is one a plugin registered, or a shared command the plugins declare handlers for. */
    isPluginCommand: (id: string): boolean =>
      commands.hasCommand(id) || commands.handlers.isSharedCommand(id),
    run: async (id: string, target: CommandTarget, run: PluginRun): Promise<boolean> =>
      commands.executeCommand(id, await getCommandNote(target), run),
    /** Offers a key to the commands of the plugins; returns whether one took it. */
    handleShortcut: (
      event: KeyboardEvent,
      getTarget: () => CommandTarget,
      runApp: (command: string, values: Values) => void,
    ): boolean =>
      commands.handleShortcut(event, () => getCommandNote(getTarget()), {
        target: toRunTarget(getTarget()),
        runApp,
      }),
  }
}
