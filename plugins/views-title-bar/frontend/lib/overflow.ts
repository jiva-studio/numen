import type { TitleBarAction } from '@numen/plugin-sdk'

export interface SplitActions {
  readonly visible: readonly TitleBarAction[]
  readonly overflow: readonly TitleBarAction[]
}

/**
 * The actions that stay in the bar and the ones that go to the menu, in the order given. When all
 * of them fit they stay; otherwise the menu button takes one of the slots.
 */
export function splitActions(actions: readonly TitleBarAction[], capacity: number): SplitActions {
  if (actions.length <= capacity) return { visible: actions, overflow: [] }
  const slots = Math.max(Math.floor(capacity) - 1, 0)
  return { visible: actions.slice(0, slots), overflow: actions.slice(slots) }
}
