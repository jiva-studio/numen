import type { CommandsHost } from '@numen/plugin-sdk'
import { getCommandOwner } from '../../commandId'
import type { RequirePermission } from './permission'

/**
 * The commands of the application that plugins run without a slot listing them, because a plugin
 * starts them from its own button: the one place to widen when a plugin needs another.
 */
const RUNNABLE_APP_COMMANDS: ReadonlySet<string> = new Set([
  'app:vault.create',
  'app:settings.open',
  'app:file.reveal',
])

/** The commands that move a view and change nothing, so a plugin that may read the vault runs them; every other command needs `vault:write`. */
const READ_ONLY_COMMANDS: ReadonlySet<string> = new Set(['app:file.reveal', 'app:settings.open'])

/** What decides whether a plugin runs a command it does not own. */
export interface CommandRunPolicy {
  readonly pluginId: string
  readonly isListed: (command: string) => boolean
  readonly getPermission?: (command: string) => string | undefined
  readonly checkPermission?: RequirePermission
}

/**
 * The commands service with a `run` that refuses a command the plugin does not own, no slot lists
 * and the short list of application commands lacks. The refusal rejects.
 */
export function limitCommands(commands: CommandsHost, policy: CommandRunPolicy): CommandsHost {
  const canRun = (command: string): boolean =>
    getCommandOwner(command) === policy.pluginId ||
    RUNNABLE_APP_COMMANDS.has(command) ||
    policy.isListed(command)
  return {
    getIds: () => commands.getIds(),
    watch: (handler) => commands.watch(handler),
    listItems: (command, target) => commands.listItems(command, target),
    run: (...args) => {
      if (!canRun(args[0])) {
        return Promise.reject(
          new Error(
            `Plugin "${policy.pluginId}" runs the command "${args[0]}", which it does not own and no slot lists`,
          ),
        )
      }
      const permissions = [
        READ_ONLY_COMMANDS.has(args[0]) ? 'vault:read' : 'vault:write',
        policy.getPermission?.(args[0]),
      ]
      try {
        for (const permission of permissions) {
          if (permission) policy.checkPermission?.(permission, 'commands.run')
        }
      } catch (error) {
        return Promise.reject(error)
      }
      return commands.run(...args)
    },
  }
}
