/**
 * The health items at error severity, as failed tasks of the list, each with the fix it offers.
 *
 * A warning shows only on the page of its plugin. An item whose command the window has no handler
 * for is listed without a fix.
 */
import type { CommandsHost } from '@numen/plugin-loader'
import type { PluginHealthItem } from '@/entities/plugin'
import type { Task } from '@/entities/window'

/** What the health tasks read of the window: its commands and their titles, and the plugin names. */
export interface HealthTaskDeps {
  readonly commands: Pick<CommandsHost, 'getIds' | 'run'>
  readonly getCommandTitle: (command: string) => string
  readonly getPluginName: (pluginId: string) => string
}

const HEALTH_TASK_PREFIX = 'health:'

const getTaskId = (item: PluginHealthItem): string =>
  `${HEALTH_TASK_PREFIX}${item.pluginId}:${item.key}`

const getActionLabel = (item: PluginHealthItem, deps: HealthTaskDeps): string => {
  const action = item.action
  if (!action || !deps.commands.getIds().includes(action.command)) return ''
  return action.title || deps.getCommandTitle(action.command)
}

/** The failed tasks of the items at error severity, in the order the core lists them. */
export function getHealthTasks(
  items: readonly PluginHealthItem[],
  deps: HealthTaskDeps,
): readonly Task[] {
  return items
    .filter((item) => item.severity === 'error')
    .map((item) => {
      const name = deps.getPluginName(item.pluginId)
      const label = getActionLabel(item, deps)
      return {
        id: getTaskId(item),
        label: name,
        about: item.detail,
        progress: { completedCount: 0, total: 0, counting: 'items' },
        error: item.text,
        source: { isUserStarted: false, pluginId: item.pluginId, pluginName: name },
        isCancellable: false,
        kind: 'work',
        action: label ? { label } : undefined,
      }
    })
}

/** Runs the fix of the item the task of the id lists; a task of no item, or of no fix, runs nothing. */
export async function runHealthAction(
  id: string,
  items: readonly PluginHealthItem[],
  deps: Pick<HealthTaskDeps, 'commands'>,
): Promise<void> {
  const action = items.find((item) => getTaskId(item) === id)?.action
  if (!action || !deps.commands.getIds().includes(action.command)) return
  await deps.commands.run(action.command, undefined, undefined, action.values)
}
