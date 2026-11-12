/**
 * The tabs a window opens again: the tabs its kinds place, the tabs the core kept, and the close
 * of all of them.
 */
import { getPanes, openTab } from '@numen/workbench'
import type { Workspace } from '@numen/workbench'
import type { AnyTabKind } from '../lib/kinds'
import type { SavedTab } from '../lib/tab'
import { createWorkspace } from '../lib/workspace'
import { createTabClosers, createTabOpeners } from './tabOperations'
import type { TabStore } from './tabStore'
import { useTabPanes } from './useTabPanes'
import type { Caught } from '@/shared/api/errors'

const comparePlacement = (one: AnyTabKind, other: AnyTabKind): number => {
  // Stryker disable next-line OptionalChaining: comparePlacement is only called on kinds filtered by kind.display.placement.
  const rank = (kind: AnyTabKind) => (kind.display?.placement?.pane === 'main' ? 0 : 1)
  // Stryker disable next-line OptionalChaining,LogicalOperator: comparePlacement is only called on kinds with placement.order defined.
  return (
    rank(one) - rank(other) ||
    (one.display?.placement?.order ?? 0) - (other.display?.placement?.order ?? 0)
  )
}

/** The error callback for a placed kind whose tab failed to open. */
type PlacementError = (kind: string, error: Caught) => void

async function createPlacedWorkspace(
  kinds: readonly AnyTabKind[],
  createTab: (kind: string) => Promise<string>,
  onError: PlacementError,
): Promise<Workspace> {
  const main: string[] = []
  const aside: string[] = []
  for (const one of kinds.filter((kind) => kind.display?.placement).sort(comparePlacement)) {
    try {
      const id = await createTab(one.kind)
      // Stryker disable next-line OptionalChaining: kinds are filtered by kind.display.placement above.
      if (one.display?.placement?.pane === 'main') main.push(id)
      else aside.push(id)
    } catch (error) {
      onError(one.kind, error)
    }
  }
  return createWorkspace(main, aside)
}

export function useRestoredTabs(store: TabStore) {
  const { open, layout, byKind } = store
  const openers = createTabOpeners(store)
  const { show } = useTabPanes(store)

  /** A kind whose tab fails to open is reported to `onError` and omitted from the layout. */
  const openPlacedTabs = async (onError: PlacementError) => {
    const earlier = getPanes(layout.value.root).flatMap((one) => one.tabs)
    layout.value = await createPlacedWorkspace(
      [...byKind.value.values()],
      openers.createTab,
      onError,
    )
    // Stryker disable next-line ArrowFunction: openTab is idempotent for tabs already placed in the workspace
    const placedItems = new Set(getPanes(layout.value.root).flatMap((one) => one.tabs))
    for (const id of earlier) {
      if (open.value.has(id) && !placedItems.has(id)) layout.value = openTab(layout.value, id)
    }
  }

  /**
   * The tabs the core kept, opened again with the state each kind saved and laid out by the
   * placement of their kind. A tab whose kind is not registered opens as a placeholder.
   * The core lists the tab shown last at the end, so the last tab of each pane is the one in front
   * there; the active tab is shown after them and takes the focus.
   */
  const restoreTabs = (saved: readonly SavedTab[], activeId: string) => {
    const main: string[] = []
    const aside: string[] = []
    for (const one of saved) {
      if (!open.value.has(one.id)) openers.openSavedTab(one)
      const kind = byKind.value.get(one.kind)
      if (kind?.display?.placement?.pane === 'aside') aside.push(one.id)
      else main.push(one.id)
    }
    layout.value = createWorkspace(main, aside)
    for (const front of [main.at(-1), aside.at(-1)]) {
      if (front) layout.value = openTab(layout.value, front)
    }
    if (open.value.has(activeId)) layout.value = openTab(layout.value, activeId)
  }

  /** A tab the core kept, opened and shown; a tab already open is only shown. */
  const openSavedTab = (saved: SavedTab) => {
    if (!open.value.has(saved.id)) openers.openSavedTab(saved)
    show(saved.id)
  }

  return {
    openPlacedTabs,
    restoreTabs,
    openSavedTab,
    close: createTabClosers(store).close,
  }
}
