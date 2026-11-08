/**
 * The items of a command, whoever owns it: a command of the window that offers them over its
 * target, or a command a plugin registered with items; and the run of one with the item picked.
 */
import {
  SHARED_COMMANDS,
  type CommandItem,
  type CommandTarget as RunTarget,
  type PluginHost,
} from '@numen/plugin-host'
import type { Command, CommandTarget } from '@/features/command-palette'
import { getNamedTarget } from './runTarget'

interface CommandItemsDeps {
  readonly getAppCommands: () => readonly Command[]
  readonly pluginHost: Pick<PluginHost, 'hasCommandItems' | 'listCommandItems'> | undefined
  readonly getTarget: () => CommandTarget
  readonly runApp: (id: string, target: CommandTarget, itemId: string) => Promise<void>
  readonly runPlugin: (id: string, target: RunTarget, itemId: string) => Promise<void>
  readonly readTarget?: (target: RunTarget) => Promise<RunTarget>
}

export function createCommandItems(deps: CommandItemsDeps) {
  const findAppCommand = (id: string) =>
    deps.getAppCommands().find((one) => one.id === id && one.listItems !== undefined)

  const toAppTarget = (target: RunTarget): CommandTarget => getNamedTarget(deps.getTarget(), target)

  return {
    hasCommandItems: (id: string): boolean =>
      findAppCommand(id) !== undefined || (deps.pluginHost?.hasCommandItems(id) ?? false),
    listCommandItems: async (id: string, target: RunTarget): Promise<readonly CommandItem[]> => {
      const command = findAppCommand(id)
      if (command) return (await command.listItems?.(toAppTarget(target))) ?? []
      if (!deps.pluginHost?.hasCommandItems(id)) return []
      const isShared = SHARED_COMMANDS.some((one) => one.id === id)
      const noteTarget = isShared && deps.readTarget ? await deps.readTarget(target) : target
      return deps.pluginHost.listCommandItems(id, noteTarget)
    },
    runWithItem: (id: string, target: RunTarget, itemId: string): Promise<void> =>
      findAppCommand(id)
        ? deps.runApp(id, toAppTarget(target), itemId)
        : deps.runPlugin(id, target, itemId),
  }
}
