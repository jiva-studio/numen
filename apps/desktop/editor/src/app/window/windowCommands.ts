/**
 * The commands the window gives its plugins: the ids of its own commands and of those its plugins
 * registered, the items a command offers, and the run of one, which settles when the run ends and
 * rejects with its error, so the plugin that runs it shows the error and the window writes none.
 */
import { computed, watch } from 'vue'
import type { CommandItem, CommandsHost, CommandTarget as RunTarget } from '@numen/plugin-host'
import type { CommandTarget } from '@/features/command-palette'

interface WindowCommandsDeps {
  readonly getAppIds: () => readonly string[]
  /** The ids of the commands the plugins of the window registered; reactive. */
  readonly getPluginIds: () => readonly string[]
  /** Runs the command over the target; settles when the run ends and rejects with its error, which it writes nowhere. */
  readonly runCommandToEnd: (id: string, target: CommandTarget) => Promise<void>
  /** The items a command offers over the target; none for a command of the window. */
  readonly listItems: (id: string, target: RunTarget) => Promise<readonly CommandItem[]>
  /** Runs a command with the item picked as its argument. */
  readonly runWithArgument: (id: string, target: RunTarget, argument: string) => Promise<void>
  readonly getTarget: () => CommandTarget
}

export function createWindowCommands(deps: WindowCommandsDeps): CommandsHost {
  const ids = computed<readonly string[]>(() => [...deps.getAppIds(), ...deps.getPluginIds()])

  /**
   * The target of the tab in front, with what the run names in place of its own. A run that names
   * no target, as from a bar, is over the file of the tab in front, named as a menu names its own.
   */
  const getRunTarget = (target: RunTarget | undefined): CommandTarget => {
    const front = deps.getTarget()
    if (!target) return { ...front, isTargetNamed: true }
    return {
      ...front,
      path: target.path,
      file: target.path,
      title: target.title,
      isFolder: target.isFolder,
      others: target.others,
      tab: target.tab || front.tab,
      isTargetNamed: true,
    }
  }

  /** The run target in the words of the SDK, which the items and a run with an argument take. */
  const toRunTarget = (target: CommandTarget): RunTarget => ({
    path: target.path,
    title: target.title,
    isFolder: target.isFolder ?? false,
    others: target.others ?? [],
    tab: target.tab,
  })

  return {
    getIds: () => ids.value,
    watch: (handler) => ({ dispose: watch(ids, handler, { immediate: true }) }),
    listItems: (id, target) => deps.listItems(id, toRunTarget(getRunTarget(target))),
    run: async (id, target, argument) => {
      const runTarget = getRunTarget(target)
      if (argument === undefined) await deps.runCommandToEnd(id, runTarget)
      else await deps.runWithArgument(id, toRunTarget(runTarget), argument)
    },
  }
}
