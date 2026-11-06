/**
 * The permission a plugin needs to run a command of the window: `vault:write`, except for the
 * commands that change nothing in the vault.
 */
import type { CommandsHost } from '@numen/plugin-sdk'
import { APP_COMMAND_OWNER, formatCommandId } from '../commandId'

/** The commands of the window that change nothing in the vault. */
export const READ_ONLY_COMMANDS: ReadonlySet<string> = new Set([
  formatCommandId(APP_COMMAND_OWNER, 'note.open'),
  formatCommandId(APP_COMMAND_OWNER, 'note.copyPath'),
  formatCommandId(APP_COMMAND_OWNER, 'settings.open'),
  formatCommandId('md.numen.views.plex.graph', 'graph.showNote'),
  formatCommandId('md.numen.views.agent.chat', 'note.ask'),
])

/** The commands of the window, run only with `vault:write` unless the command changes nothing. */
export function guardCommands(
  commands: CommandsHost,
  requireWrite: (operation: string) => void,
): CommandsHost {
  return {
    getIds: () => commands.getIds(),
    watch: (handler) => commands.watch(handler),
    run: async (id, target) => {
      if (!READ_ONLY_COMMANDS.has(id)) requireWrite(`commands.run ${id}`)
      await commands.run(id, target)
    },
  }
}
