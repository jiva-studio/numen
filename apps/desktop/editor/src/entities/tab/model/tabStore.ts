/**
 * What the parts of a window's tabs share: the tabs that are open, how they are laid out in
 * panes, and the kinds that can open one.
 */
import { ref, shallowRef, type EffectScope, type Ref, type ShallowRef } from 'vue'
import { openTab, pane } from '@numen/workbench'
import type { Workspace } from '@numen/workbench'
import type { AnyTabKind, WindowTab } from '../lib/kinds'
import { browserTabIds, type TabIds } from '../api/tabIds'
import type { PlaceholderPanes, UnavailableServices } from './unavailable'

/**
 * The state every part of a window's tabs reads and changes. `drop` stops the effects of a tab,
 * and `unavailable` is what the placeholder of a missing kind reads and does.
 */
export interface TabStore {
  readonly open: ShallowRef<ReadonlyMap<string, WindowTab>>
  readonly layout: Ref<Workspace>
  readonly byKind: ShallowRef<ReadonlyMap<string, AnyTabKind>>
  readonly scopes: Map<string, EffectScope>
  readonly drop: (id: string) => void
  readonly unavailable: UnavailableServices
  readonly panes: PlaceholderPanes
  readonly ids: TabIds
}

export function createTabStore(
  panes: PlaceholderPanes,
  unavailable: UnavailableServices,
  ids: TabIds = browserTabIds,
): TabStore {
  const scopes = new Map<string, EffectScope>()
  const drop = (id: string) => {
    scopes.get(id)?.stop()
    // Stryker disable next-line CallExpression: a stopped scope left in the map is stopped again as a no-op
    scopes.delete(id)
  }
  return {
    open: shallowRef<ReadonlyMap<string, WindowTab>>(new Map()),
    layout: ref<Workspace>({ root: pane('main', []), axis: 'horizontal', focus: 'main' }),
    byKind: shallowRef<ReadonlyMap<string, AnyTabKind>>(new Map()),
    scopes,
    drop,
    unavailable,
    panes,
    ids,
  }
}

/** The path a tab reports, else the one its target names. */
export const getTabPath = (tab: WindowTab): string => {
  const address = tab.kind.address
  const reportedPath = address?.getOpenTab?.(tab.state).path
  if (reportedPath) return reportedPath
  const target = address?.getTarget?.(tab.state)
  // Stryker disable next-line StringLiteral: reconcileTab returns before comparing when the requested path is empty, so the empty fallback is never compared
  return target?.path || target?.file || ''
}

/** Runs what opens a tab in an effect scope, and stops the scope where it throws. */
export const openInScope = <T extends object>(scope: EffectScope, open: () => T): T => {
  try {
    const state = scope.run(open)
    if (state === undefined) throw new Error('the scope of the tab is not active')
    return state
  } catch (error) {
    scope.stop()
    throw error
  }
}

/** The map without the entry. */
export const omitEntry = <T>(
  target: ReadonlyMap<string, T>,
  id: string,
): ReadonlyMap<string, T> => {
  const rest = new Map(target)
  rest.delete(id)
  return rest
}

/** Activates an open tab in its pane. */
export const showTab = (store: TabStore, id: string): void => {
  store.layout.value = openTab(store.layout.value, id)
}
