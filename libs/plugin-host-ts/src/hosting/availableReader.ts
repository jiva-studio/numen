import {
  getIssueKey,
  getIssueMessage,
  listAvailableCommands,
  type AvailableCommand,
} from './available'
import type { HostRegistry } from './registry'
import type { PluginHostDeps } from './types'

/**
 * Reads the listed commands. Each issue found is reported once through `deps.reportCommandIssue`,
 * keyed by its plugin, id and kind.
 */
export function readAvailableCommands(
  deps: PluginHostDeps,
  registry: HostRegistry,
): AvailableCommand[] {
  const { commands, issues } = listAvailableCommands(deps, registry)
  for (const issue of issues) {
    const key = getIssueKey(issue)
    if (registry.reportedIssues.has(key)) continue
    registry.reportedIssues.add(key)
    deps.reportCommandIssue?.(issue.pluginId, getIssueMessage(issue))
  }
  return commands
}
