/**
 * The items of a command, whoever owns it: a command of the window that offers them over its
 * target, or a command a plugin registered with items; and the run of one with the item picked.
 */
import {
  isSharedCommand,
  type CommandItem,
  type CommandTarget as RunTarget,
  type PluginHost,
} from '@numen/plugin-loader'
import type { Command, CommandTarget } from '@/entities/command'
import { getNamedTarget } from './runTarget'

type Values = Readonly<Record<string, string>>

interface CommandItemsDeps {
  readonly getAppCommands: () => readonly Command[]
  readonly pluginHost: Pick<PluginHost, 'hasCommandItems' | 'listCommandItems'> | undefined
  readonly getTarget: () => CommandTarget
  readonly runApp: (
    id: string,
    target: CommandTarget,
    itemId: string,
    values?: Values,
  ) => Promise<void>
  readonly runPlugin: (
    id: string,
    target: RunTarget,
    itemId: string,
    values?: Values,
  ) => Promise<void>
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
      const noteTarget =
        isSharedCommand(id) && deps.readTarget ? await deps.readTarget(target) : target
      return deps.pluginHost.listCommandItems(id, noteTarget)
    },
    runWithItem: (
      id: string,
      target: RunTarget,
      itemId: string,
      values?: Values,
    ): Promise<void> => {
      const extra = values === undefined ? [] : [values]
      return findAppCommand(id)
        ? deps.runApp(id, toAppTarget(target), itemId, ...extra)
        : deps.runPlugin(id, target, itemId, ...extra)
    },
  }
}
