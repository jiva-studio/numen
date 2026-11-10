import type { CommandsHost } from '@numen/plugin-sdk'
import { getCommandOwner } from '../../commandId'

/**
 * The commands of the application that plugins run without a slot listing them, because a plugin
 * starts them from its own button: the one place to widen when a plugin needs another.
 */
const RUNNABLE_APP_COMMANDS: ReadonlySet<string> = new Set([
  'app:vault.create',
  'app:settings.open',
])

/** What decides whether a plugin runs a command it does not own. */
export interface CommandRunPolicy {
  readonly pluginId: string
  /** Whether some manifest places the command in a slot, which shows the user a row for it. */
  readonly isListed: (command: string) => boolean
}

/**
 * The commands service with a `run` that refuses a command nobody offers: a plugin runs its own
 * commands, the commands a slot lists and the short list of application commands, and nothing
 * else. The refusal rejects.
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
    run: (command, target, itemId, values) =>
      canRun(command)
        ? commands.run(command, target, itemId, values)
        : Promise.reject(
            new Error(
              `Plugin "${policy.pluginId}" runs the command "${command}", which it does not own and no slot lists`,
            ),
          ),
  }
}
