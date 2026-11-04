import { getIssueKey, getIssueMessage, listOfferedCommands, type OfferedCommand } from './offered'
import type { HostRegistry } from './registry'
import type { PluginHostDeps } from './types'

/**
 * Reads the offered commands. Each issue found is reported once through `deps.reportCommandIssue`,
 * keyed by its plugin, id and kind.
 */
export function readOfferedCommands(
  deps: PluginHostDeps,
  registry: HostRegistry,
): OfferedCommand[] {
  const { commands, issues } = listOfferedCommands(deps, registry)
  for (const issue of issues) {
    const key = getIssueKey(issue)
    if (registry.reportedIssues.has(key)) continue
    registry.reportedIssues.add(key)
    deps.reportCommandIssue?.(issue.pluginId, getIssueMessage(issue))
  }
  return commands
}
