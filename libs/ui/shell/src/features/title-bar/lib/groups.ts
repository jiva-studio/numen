import type { TitleBarAction } from '../model/types'

/** The ids of the actions that begin a group after the first one, where a separator is drawn. */
export function findGroupStarts(actions: readonly TitleBarAction[]): ReadonlySet<string> {
  const starts = new Set<string>()
  actions.forEach((action, index) => {
    const before = actions[index - 1]
    if (before && before.group !== action.group) starts.add(action.id)
  })
  return starts
}
