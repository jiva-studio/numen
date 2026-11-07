/**
 * The items of a command, whoever owns it: a command of the window that offers them over its
 * target, or a command a plugin registered with items; and the run of one with the argument picked.
 */
import {
  SHARED_COMMANDS,
  type CommandItem,
  type CommandTarget as RunTarget,
  type PluginHost,
} from '@numen/plugin-host'
import type { Command, CommandTarget } from '@/features/command-palette'

interface CommandItemsDeps {
  /** The commands of the window that may offer items. */
  readonly getAppCommands: () => readonly Command[]
  readonly pluginHost: Pick<PluginHost, 'hasCommandItems' | 'listCommandItems'> | undefined
  /** The target of the tab in front, which a command of the window runs over by default. */
  readonly getTarget: () => CommandTarget
  /** Runs a command of the window with the argument picked. */
  readonly runApp: (id: string, target: CommandTarget, argument: string) => Promise<void>
  /** Runs a plugin command with the argument picked. */
  readonly runPlugin: (id: string, target: RunTarget, argument: string) => Promise<void>
  /** The target with the note type and properties of its file, which the handlers of a shared command read. */
  readonly readTarget?: (target: RunTarget) => Promise<RunTarget>
}

export function createCommandItems(deps: CommandItemsDeps) {
  const findAppCommand = (id: string) =>
    deps.getAppCommands().find((one) => one.id === id && one.listItems !== undefined)

  /** The target of the tab in front, with what the run names in place of its own. */
  const toAppTarget = (target: RunTarget): CommandTarget => {
    const front = deps.getTarget()
    return {
      ...front,
      path: target.path,
      file: target.path === '' ? front.file : target.path,
      title: target.title,
      isFolder: target.isFolder,
      others: target.others,
      tab: target.tab || front.tab,
    }
  }

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
    runWithArgument: (id: string, target: RunTarget, argument: string): Promise<void> =>
      findAppCommand(id)
        ? deps.runApp(id, toAppTarget(target), argument)
        : deps.runPlugin(id, target, argument),
  }
}
