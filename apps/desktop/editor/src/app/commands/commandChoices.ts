/**
 * The choices of a command, whoever owns it: a command of the window that offers them over its
 * target, or a command a plugin registered with choices; and the run of one with the choice picked.
 */
import type { CommandChoice, CommandTarget as RunTarget, PluginHost } from '@numen/plugin-host'
import type { Command, CommandTarget } from '@/features/command-palette'

interface CommandChoicesDeps {
  /** The commands of the window that may offer choices. */
  readonly getAppCommands: () => readonly Command[]
  readonly pluginHost: Pick<PluginHost, 'hasCommandChoices' | 'getCommandChoices'> | undefined
  /** The target of the tab in front, which a command of the window runs over by default. */
  readonly getTarget: () => CommandTarget
  /** Runs a command of the window with the choice picked. */
  readonly runApp: (id: string, target: CommandTarget, choice: string) => Promise<void>
  /** Runs a plugin command with the choice picked. */
  readonly runPlugin: (id: string, target: RunTarget, choice: string) => Promise<void>
}

export function createCommandChoices(deps: CommandChoicesDeps) {
  const findAppCommand = (id: string) =>
    deps.getAppCommands().find((one) => one.id === id && one.getChoices !== undefined)

  /** The target of the tab in front, with what the run names in place of its own. */
  const toAppTarget = (target: RunTarget): CommandTarget => {
    const front = deps.getTarget()
    return {
      ...front,
      path: target.path,
      title: target.title,
      isFolder: target.isFolder,
      others: target.others,
      tab: target.tab || front.tab,
    }
  }

  return {
    hasCommandChoices: (id: string): boolean =>
      findAppCommand(id) !== undefined || (deps.pluginHost?.hasCommandChoices(id) ?? false),
    getCommandChoices: async (id: string, target: RunTarget): Promise<readonly CommandChoice[]> => {
      const command = findAppCommand(id)
      if (command) return command.getChoices?.(toAppTarget(target)) ?? []
      return deps.pluginHost?.hasCommandChoices(id)
        ? deps.pluginHost.getCommandChoices(id, target)
        : []
    },
    runChoice: (id: string, target: RunTarget, choice: string): Promise<void> =>
      findAppCommand(id)
        ? deps.runApp(id, toAppTarget(target), choice)
        : deps.runPlugin(id, target, choice),
  }
}

export type CommandChoices = ReturnType<typeof createCommandChoices>
