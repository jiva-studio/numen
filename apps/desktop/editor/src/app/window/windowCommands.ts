/**
 * The commands the window gives its plugins: the ids of its own commands and of those its plugins
 * registered, the items a command offers, and the run of one, which settles when the run ends and
 * rejects with its error, so the plugin that runs it shows the error and the window writes none.
 */
import { computed, watch } from 'vue'
import type { CommandItem, CommandsHost, CommandTarget as RunTarget } from '@numen/plugin-host'
import type { CommandTarget } from '@/features/command-palette'
import { getNamedTarget, toRunTarget } from '../commands/runTarget'

interface WindowCommandsDeps {
  readonly getAppIds: () => readonly string[]
  readonly getPluginIds: () => readonly string[]
  readonly runCommandToEnd: (id: string, target: CommandTarget) => Promise<void>
  readonly listItems: (id: string, target: RunTarget) => Promise<readonly CommandItem[]>
  readonly runWithItem: (id: string, target: RunTarget, itemId: string) => Promise<void>
  readonly getTarget: () => CommandTarget
}

export function createWindowCommands(deps: WindowCommandsDeps): CommandsHost {
  const ids = computed<readonly string[]>(() => [...deps.getAppIds(), ...deps.getPluginIds()])

  const getRunTarget = (target: RunTarget | undefined): CommandTarget =>
    getNamedTarget(deps.getTarget(), target)

  return {
    getIds: () => ids.value,
    watch: (handler) => ({ dispose: watch(ids, handler, { immediate: true }) }),
    listItems: (id, target) => deps.listItems(id, toRunTarget(getRunTarget(target))),
    run: async (id, target, itemId) => {
      const runTarget = getRunTarget(target)
      if (itemId === undefined) await deps.runCommandToEnd(id, runTarget)
      else await deps.runWithItem(id, toRunTarget(runTarget), itemId)
    },
  }
}
