/**
 * What is done to the tabs of a window, over the state they share. Each function holds no state
 * of its own, so a part that needs an operation makes it from the store.
 */
import { effectScope } from 'vue'
import type { JsonValue } from '@/shared/api/json'
import { closeTab, openTabAdjacent, paneById } from '@numen/workbench'
import type { ActiveTab, AnyTabKind, KindTab, WindowHandle, WindowTab } from '../lib/kinds'
import type { SavedTab } from '../lib/tab'
import { createRestoreFailedKind } from './restoreFailed'
import { createUnavailableKind, isUnavailableKind, type UnavailableState } from './unavailable'
import { getTabPath, omitEntry, openInScope, showTab, type TabStore } from './tabStore'

const destroyTab = (one: WindowTab, id: string): void => {
  const events = one.kind.events
  if (events?.onDestroy) events.onDestroy(one.state, id)
  else events?.onClose?.(one.state, id)
}

export function createPlaceholders(store: TabStore) {
  /** One tab replaced by a tab of another kind opened on the same path with the same state. */
  const replaceTab = (id: string, kind: AnyTabKind, path: string, saved: JsonValue | undefined) => {
    store.drop(id)
    const scope = effectScope(true)
    const state = openInScope(scope, () => kind.open(path, id, saved))
    // Stryker disable next-line CallExpression: scope is registered for cleanup on subsequent tab close
    store.scopes.set(id, scope)
    store.open.value = new Map(store.open.value).set(id, { kind, state })
  }

  const closeTabsOfPlugin = (pluginId: string) => {
    for (const [id, one] of store.open.value) {
      if (one.kind.pluginId !== pluginId) continue
      const saved = one.kind.address?.getState?.(one.state)
      const title = one.kind.getTitle?.(one.state)
      const icon = one.kind.display?.icon
      const path = getTabPath(one)
      destroyTab(one, id)
      replaceTab(
        id,
        createUnavailableKind(
          store.panes.unavailable,
          one.kind.kind,
          { title, icon },
          store.unavailable,
        ),
        path,
        saved,
      )
    }
  }

  const reopenTabsOf = (kind: AnyTabKind) => {
    for (const [id, one] of store.open.value) {
      if (one.kind.kind !== kind.kind || !isUnavailableKind(one.kind)) continue
      const placeholder = one.state as UnavailableState
      replaceTab(id, kind, placeholder.targetPath, placeholder.saved)
    }
  }

  return { closeTabsOfPlugin, reopenTabsOf }
}

export function createTabQueries(store: TabStore) {
  const { open, layout } = store
  const getTab = (id: string) => open.value.get(id) ?? null

  const getTabStateIn = <T>(id: string, kind: string): T | null => {
    const one = open.value.get(id)
    return one && one.kind.kind === kind ? (one.state as T) : null
  }

  const front = (): ActiveTab | null => {
    const id = paneById(layout.value.root, layout.value.focus)?.active
    if (!id) return null
    const one = open.value.get(id)
    return one ? { id, kind: one.kind.kind, state: one.state } : { id, kind: null, state: null }
  }

  const each = <T>(kind: string): readonly KindTab<T>[] =>
    [...open.value]
      .filter(([, one]) => one.kind.kind === kind)
      .map(([id, one]) => ({ id, state: one.state as T }))

  return { getTab, getTabStateIn, front, each }
}

const openInIsolation = (store: TabStore, saved: SavedTab, one: AnyTabKind) => {
  const scope = effectScope(true)
  try {
    return [
      one,
      scope,
      openInScope(scope, () => one.open(saved.path, saved.id, saved.saved)),
    ] as const
  } catch (error) {
    const failed = createRestoreFailedKind(
      store.panes.restoreFailed,
      saved.kind,
      saved.title,
      error,
    )
    // Stryker disable next-line BooleanLiteral: detached scope flag prevents parent scope collection
    const placeholderScope = effectScope(true)
    return [
      failed,
      placeholderScope,
      openInScope(placeholderScope, () => failed.open(saved.path, saved.id, saved.saved)),
    ] as const
  }
}

export function createTabOpeners(store: TabStore) {
  /** A tab the core kept, opened under its id with the state its kind saved. */
  const openSavedTab = (saved: SavedTab) => {
    const one =
      store.byKind.value.get(saved.kind) ??
      createUnavailableKind(
        store.panes.unavailable,
        saved.kind,
        { title: saved.title },
        store.unavailable,
      )
    const [kind, scope, state] = openInIsolation(store, saved, one)
    store.scopes.set(saved.id, scope)
    store.open.value = new Map(store.open.value).set(saved.id, { kind, state })
  }

  const findOpenTabAt = (kind: string, targetPath: string): string | undefined => {
    for (const [existingId, existingTab] of store.open.value) {
      if (existingTab.kind.kind === kind && getTabPath(existingTab) === targetPath) {
        return existingId
      }
    }
    return undefined
  }

  const getKind = (kind: string): AnyTabKind =>
    store.byKind.value.get(kind) ??
    createUnavailableKind(store.panes.unavailable, kind, undefined, store.unavailable)

  /** The open tab a path already has, where the kind gives a tab one identity per path. */
  const findTabOfPath = (one: AnyTabKind, kind: string, targetPath: string) =>
    // Stryker disable next-line LogicalOperator: empty targetPath matches singleton tabs via candidateId below
    targetPath && one.address?.identity ? findOpenTabAt(kind, targetPath) : undefined

  const getCandidateId = (one: AnyTabKind, kind: string, targetPath: string, wantedId: string) =>
    wantedId ||
    (one.address?.identity
      ? `${kind}:${one.address.identity(targetPath)}`
      : store.ids.createTabId(kind))

  const createTabNow = (kind: string, targetPath: string, wantedId = ''): string => {
    const one = getKind(kind)
    const matchedId = findTabOfPath(one, kind, targetPath)
    if (matchedId) return matchedId
    const candidateId = getCandidateId(one, kind, targetPath, wantedId)
    const existing = store.open.value.get(candidateId)
    if (existing && (!targetPath || !getTabPath(existing))) return candidateId
    const id = existing ? store.ids.createTabId(kind) : candidateId
    const scope = effectScope(true)
    const state = openInScope(scope, () => one.open(targetPath, id))
    store.scopes.set(id, scope)
    store.open.value = new Map(store.open.value).set(id, { kind: one, state })
    return id
  }

  const createTab = (kind: string, targetPath = ''): Promise<string> =>
    new Promise((resolve) => {
      resolve(createTabNow(kind, targetPath))
    })

  /** Opens a tab of a kind and shows it. A non-empty `id` reuses a tab id from another window. */
  const openTabOfKind = (kind: string, target = '', id = ''): Promise<string> =>
    new Promise((resolve) => {
      const tabId = createTabNow(kind, target, id)
      showTab(store, tabId)
      resolve(tabId)
    })

  const openTabAdjacentOfKind = (kind: string, target = ''): Promise<string> =>
    new Promise((resolve) => {
      const id = createTabNow(kind, target)
      store.layout.value = openTabAdjacent(store.layout.value, id, {
        side: 'right',
        createId: store.ids.createPaneId,
      })
      resolve(id)
    })

  return {
    createTab,
    openSavedTab,
    openTabOfKind,
    openTabAdjacentOfKind,
  }
}

export function createTabClosers(store: TabStore) {
  const show = (id: string) => {
    showTab(store, id)
  }

  const finishClose = (id: string) => {
    store.open.value = omitEntry(store.open.value, id)
    store.drop(id)
    store.layout.value = closeTab(store.layout.value, id)
  }

  const onTabShown = (id: string) => {
    const one = store.open.value.get(id)
    if (!one) return
    store.open.value = new Map([...omitEntry(store.open.value, id), [id, one]])
    one.kind.display?.onShow?.(one.state, id)
  }

  const releaseTab = (id: string): boolean => {
    const one = store.open.value.get(id)
    if (!one) return true
    if (one.kind.events?.onClose?.(one.state, id) === false) return false
    store.open.value = omitEntry(store.open.value, id)
    store.drop(id)
    return true
  }

  const requestClose = (id: string) => {
    if (releaseTab(id)) store.layout.value = closeTab(store.layout.value, id)
  }

  const reconcileTab = (path: string, newKind: string) => {
    if (!path) return
    for (const [id, one] of store.open.value) {
      if (one.kind.kind === newKind || one.kind.display?.isTool) continue
      if (getTabPath(one) === path) requestClose(id)
    }
  }

  const close = () => {
    for (const [id, one] of store.open.value) {
      destroyTab(one, id)
      store.drop(id)
    }
  }

  return { show, finishClose, onTabShown, releaseTab, requestClose, reconcileTab, close }
}

export function createTabNavigation(
  store: TabStore,
  openers: Pick<ReturnType<typeof createTabOpeners>, 'openTabOfKind'>,
) {
  const { open } = store
  /**
   * Brings the most recently active tab of a kind to the front and navigates it
   * to the path; opens a tab of the kind at the path when none is open.
   */
  const navigateTab = async (kind: string, targetPath: string): Promise<string> => {
    const last = [...open.value].filter(([, one]) => one.kind.kind === kind).at(-1)
    if (!last) return openers.openTabOfKind(kind, targetPath)
    const [id, one] = last
    showTab(store, id)
    await one.kind.address?.navigate?.(one.state, targetPath, '')
    return id
  }

  /**
   * Every tab standing on a path, of a kind that navigates, moved to another.
   * Calls onMove with the id of each tab before it navigates.
   */
  const navigateTabsAt = async (
    from: string,
    to: string,
    onMove?: (tabId: string) => void,
  ): Promise<void> => {
    const matchingTabs = [...open.value].filter(
      ([, one]) => one.kind.address?.navigate && getTabPath(one) === from,
    )
    await Promise.all(
      matchingTabs.map(([id, one]) => {
        onMove?.(id)
        // Stryker disable next-line OptionalChaining: matchingTabs is filtered by one.kind.address.navigate above.
        return Promise.resolve(one.kind.address?.navigate?.(one.state, to, ''))
      }),
    )
  }

  /**
   * Sends text to the most recently active tab of a kind and brings it to the
   * front; opens a tab of the kind to receive it when none is open.
   */
  const sendText = async (kind: string, text: string): Promise<string> => {
    const last = [...open.value].filter(([, one]) => one.kind.kind === kind).at(-1)
    const id = last?.[0] ?? (await openers.openTabOfKind(kind))
    const one = open.value.get(id)
    // Stryker disable next-line OptionalChaining: id is either in open.value or just created by openTabOfKind.
    one?.kind.events?.receiveText?.(one.state, text)
    showTab(store, id)
    return id
  }

  return { navigateTab, navigateTabsAt, sendText }
}

export function createWindowHandle(store: TabStore): WindowHandle {
  const openers = createTabOpeners(store)
  const queries = createTabQueries(store)
  const closers = createTabClosers(store)
  return {
    openTab: (kind, target) => openers.openTabOfKind(kind, target),
    openTabAdjacent: (kind, target) => openers.openTabAdjacentOfKind(kind, target),
    show: (id) => {
      if (queries.getTab(id)) closers.show(id)
    },
    closeTab: (id) => closers.finishClose(id),
    each: <T>(kind: string) => queries.each<T>(kind),
    last: <T>(kind: string) => queries.each<T>(kind).at(-1) ?? null,
    front: () => queries.front(),
    getTabState: <T>(kind: string, id: string) => queries.getTabStateIn<T>(id, kind),
  }
}
