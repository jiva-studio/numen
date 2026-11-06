/**
 * The commands the window gives its plugins: the ids of its own commands and of those its plugins
 * registered, the choices a command offers, and the run of one through the runner the palette and
 * the shortcuts use.
 */
import { computed, watch } from 'vue'
import type { CommandChoice, CommandsHost, CommandTarget as RunTarget } from '@numen/plugin-host'
import type { CommandTarget } from '@/features/command-palette'

interface WindowCommandsDeps {
  readonly getAppIds: () => readonly string[]
  /** The ids of the commands the plugins of the window registered; reactive. */
  readonly getPluginIds: () => readonly string[]
  /** The runner the palette and the shortcuts use. */
  readonly runCommand: (id: string, target: CommandTarget) => void
  /** The choices a command offers over the target; none for a command of the window. */
  readonly getChoices: (id: string, target: RunTarget) => Promise<readonly CommandChoice[]>
  /** Runs a command with the choice picked. */
  readonly runChoice: (id: string, target: RunTarget, choice: string) => Promise<void>
  readonly getTarget: () => CommandTarget
}

export function createWindowCommands(deps: WindowCommandsDeps): CommandsHost {
  const ids = computed<readonly string[]>(() => [...deps.getAppIds(), ...deps.getPluginIds()])

  /** The target of the tab in front, with what the run names in place of its own. */
  const getRunTarget = (target: RunTarget | undefined): CommandTarget => {
    const front = deps.getTarget()
    if (!target) return front
    return {
      ...front,
      path: target.path,
      title: target.title,
      isFolder: target.isFolder,
      others: target.others,
      tab: target.tab || front.tab,
    }
  }

  /** The run target in the words of the SDK, which the choices and a run with a choice take. */
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
    getChoices: (id, target) => deps.getChoices(id, toRunTarget(getRunTarget(target))),
    run: async (id, target, choice) => {
      const runTarget = getRunTarget(target)
      if (choice === undefined) deps.runCommand(id, runTarget)
      else await deps.runChoice(id, toRunTarget(runTarget), choice)
    },
  }
}
