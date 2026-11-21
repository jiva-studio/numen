import type { PluginContribution, PluginContributionKind } from './pluginInfo'

const SUMMARIZED: ReadonlyArray<readonly [PluginContributionKind, string, string]> = [
  ['tools', 'tool', 'tools'],
  ['searchFunctions', 'search function', 'search functions'],
]

/** The count of the tools and search functions a plugin contributes, such as `2 tools`. Empty for none. */
export function getContributionSummary(
  contributions: readonly PluginContribution[] | undefined,
): string {
  return SUMMARIZED.flatMap(([kind, one, many]) => {
    const count = contributions?.find((group) => group.kind === kind)?.items.length ?? 0
    return count === 0 ? [] : [`${count} ${count === 1 ? one : many}`]
  }).join(', ')
}
