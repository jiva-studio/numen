/**
 * The commands the window gives its plugins: the ids of its own commands and of those its plugins
 * registered, the items a command offers, and the run of one, which settles when the run ends and
 * rejects with its error, so the plugin that runs it shows the error and the window writes none.
 */
import { computed, watch } from 'vue'
import type { CommandItem, CommandsHost, CommandTarget as RunTarget } from '@numen/plugin-loader'
import type { CommandTarget } from '@/shared/model/command'
import { getNamedTarget, toRunTarget } from '../../lib/runTarget'

type Values = Readonly<Record<string, string>>

interface WindowCommandsDeps {
  readonly getAppIds: () => readonly string[]
  readonly getPluginIds: () => readonly string[]
  readonly runCommandToEnd: (id: string, target: CommandTarget, values?: Values) => Promise<string>
  readonly listItems: (id: string, target: RunTarget) => Promise<readonly CommandItem[]>
  readonly runWithItem: (
    id: string,
    target: RunTarget,
    itemId: string,
    values?: Values,
  ) => Promise<void>
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
    run: async (id, target, itemId, values) => {
      const runTarget = getRunTarget(target)
      const extra = values === undefined ? [] : [values]
      if (itemId === undefined) return deps.runCommandToEnd(id, runTarget, ...extra)
      await deps.runWithItem(id, toRunTarget(runTarget), itemId, ...extra)
      return ''
    },
  }
}
