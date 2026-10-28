/**
 * The window as a set of kinds, and the tabs it keeps of each.
 *
 * A kind says how one of its tabs opens, what it is called, what is drawn in
 * it and what letting go of it evaluates to. Everything here is written once for
 * all of them: a kind is added by declaring one, and the window is not told
 * about it twice.
 */
import {
  computed,
  effectScope,
  ref,
  shallowRef,
  type EffectScope,
  type Ref,
  type ShallowRef,
} from 'vue'
import type { Result } from '@numen/plugin-sdk'
import { closeTab, openTab, openTabBeside, pane, paneById, getPanes } from '@numen/shell'
import type { Tab, Workspace } from '@numen/shell'
import type { ActiveTab, AnyTabKind, KindTab, WindowHandle, WindowTab } from '../lib/kinds'
import { createWorkspace, generateId } from '../lib/workspace'
import { createUnavailableKind, isUnavailableKind, type UnavailableState } from '../lib/unavailable'

/** The identity a pane made by a split is filed under. */
const generatePaneId = () => crypto.randomUUID()

/** The file path a tab relies on. */
const getTabPath = (tab: WindowTab): string => {
  const openTab = tab.kind.getOpenTab?.(tab.state)
  if (openTab?.path) return openTab.path
  const target = tab.kind.getTarget?.(tab.state)
  // Stryker disable next-line StringLiteral: reconcileTab returns before comparing when the path asked for is empty, so the empty fallback is never compared
  return target?.path || target?.file || ''
}

/** A tab kept by the core, to open again. */
export interface SavedTab {
  readonly id: string
  readonly kind: string
  readonly path: string
  /** What the kind saved, parsed from its JSON; absent where it saved nothing. */
  readonly saved?: unknown
}

export function useWindowTabs() {
  const { scopes, drop } = createTabScopes()
  const open = shallowRef<ReadonlyMap<string, WindowTab>>(new Map())
  const layout = ref<Workspace>({
    root: pane('main', []),
    axis: 'horizontal',
    focus: 'main',
  })

  const tabs = computed<readonly Tab[]>(() =>
    [...open.value].map(([id, one]): Tab => {
      const mark = one.kind.getMark?.(one.state)
      const title = one.kind.getTitle?.(one.state) ?? ''
      return { id, title, ...(mark ? { mark } : {}) }
    }),
  )

  const { byKind, registerKinds, hasKind, findKind } = createTabRegistry()
  const placeholders = createPlaceholders({ open, byKind, scopes, drop })
  const registerKind = (kind: AnyTabKind): (() => void) => {
    byKind.set(kind.kind, kind)
    placeholders.reopenTabsOf(kind)
    return () => {
      if (byKind.get(kind.kind) === kind) byKind.delete(kind.kind)
    }
  }
  const queries = createTabQueries(open, layout)
  const openers = createTabOpeners({ open, layout, byKind, scopes, drop })
  const closers = createTabClosers({ open, layout, drop })
  const handle = createWindowHandle({ ...openers, ...queries })
  const navigation = createTabNavigation(open, openers)
  const request = createKindQuestions(byKind)
  /** A kind whose tab fails to open is told to `onError` and left out of the layout. */
  const openPlacedTabs = async (onError: PlacementError) => {
    const earlier = getPanes(layout.value.root).flatMap((one) => one.tabs)
    layout.value = await createPlacedWorkspace([...byKind.values()], openers.createTab, onError)
    const placedItems = new Set(getPanes(layout.value.root).flatMap((one) => one.tabs))
    for (const id of earlier) {
      if (open.value.has(id) && !placedItems.has(id)) layout.value = openTab(layout.value, id)
    }
  }
  /**
   * The tabs the core kept, opened again with the state each kind saved and laid out by the
   * placement of their kind. A tab whose kind is not registered opens as a placeholder.
   */
  const restoreTabs = (saved: readonly SavedTab[], activeId: string) => {
    const main: string[] = []
    const aside: string[] = []
    for (const one of saved) {
      if (!open.value.has(one.id)) openers.openSavedTab(one)
      const kind = byKind.get(one.kind)
      if (kind?.placement?.pane === 'aside') aside.push(one.id)
      else main.push(one.id)
    }
    layout.value = createWorkspace(main, aside)
    if (open.value.has(activeId)) layout.value = openTab(layout.value, activeId)
  }
  const onKeyPress = (event: KeyboardEvent): boolean =>
    dispatchTabKeyPress(layout.value, open.value, event)

  const api = {
    layout,
    tabs,
    handle,
    registerKinds,
    registerKind,
    hasKind,
    findKind,
    openPlacedTabs,
    restoreTabs,
    closeTabsOfPlugin: placeholders.closeTabsOfPlugin,
    createFileOfKind: request.createFileOfKind,
    navigateTab: navigation.navigateTab,
    navigateTabsAt: navigation.navigateTabsAt,
    sendText: navigation.sendText,
    getTab: queries.getTab,
    getTabStateIn: queries.getTabStateIn,
    createTab: openers.createTab,
    openTabOfKind: openers.openTabOfKind,
    openTabBesideOfKind: openers.openTabBesideOfKind,
    show: openers.show,
    finishClose: openers.finishClose,
    onTabShown: openers.onTabShown,
    onKeyPress,
    releaseTab: closers.releaseTab,
    requestClose: closers.requestClose,
    reconcileTab: closers.reconcileTab,
    close: closers.close,
  }
  return api
}

/** Registry of tab kinds supported by the window. */
function createTabRegistry() {
  const byKind = new Map<string, AnyTabKind>()
  const registerKinds = (kinds: readonly AnyTabKind[]) => {
    for (const one of kinds) byKind.set(one.kind, one)
  }
  const hasKind = (kind: string): boolean => byKind.has(kind)
  /** The first kind registered that answers yes, and nothing where none does. */
  const findKind = (isWanted: (kind: AnyTabKind) => boolean): AnyTabKind | undefined =>
    [...byKind.values()].find(isWanted)
  return { byKind, registerKinds, hasKind, findKind }
}

/**
 * The tabs of a plugin turned into placeholders while it is disabled, and back into tabs of their
 * kind once the kind is registered again. The tab keeps its id, its place and the saved state.
 */
function createPlaceholders(ctx: Omit<TabOpenerContext, 'layout'>) {
  /** One tab replaced by a tab of another kind opened on the same path with the same state. */
  const replaceTab = (id: string, kind: AnyTabKind, path: string, saved: unknown) => {
    ctx.drop(id)
    const scope = effectScope(true)
    const state = openInScope(scope, () => kind.open(path, id, saved))
    ctx.scopes.set(id, scope)
    ctx.open.value = new Map(ctx.open.value).set(id, { kind, state })
  }

  const closeTabsOfPlugin = (pluginId: string) => {
    for (const [id, one] of ctx.open.value) {
      if (one.kind.pluginId !== pluginId) continue
      const saved = one.kind.getState?.(one.state)
      const path = getTabPath(one)
      if (one.kind.onDestroy) one.kind.onDestroy(one.state, id)
      else one.kind.onClose?.(one.state, id)
      replaceTab(id, createUnavailableKind(one.kind.kind), path, saved)
    }
  }

  const reopenTabsOf = (kind: AnyTabKind) => {
    for (const [id, one] of ctx.open.value) {
      if (one.kind.kind !== kind.kind || !isUnavailableKind(one.kind)) continue
      const placeholder = one.state as UnavailableState
      replaceTab(id, kind, placeholder.targetPath, placeholder.saved)
    }
  }

  return { closeTabsOfPlugin, reopenTabsOf }
}

/** What a window asks of the kinds it holds about a file. */
function createKindQuestions(byKind: ReadonlyMap<string, AnyTabKind>) {
  /** A file made as the kind given edits it, and nothing where no kind of that name makes one. */
  const createFileOfKind = (
    kind: string,
    folder: string,
    name: string,
  ): Promise<Result<{ path: string }, string>> | undefined =>
    byKind.get(kind)?.createFile?.(folder, name)

  return { createFileOfKind }
}

/** The pane a placed kind opens in, the main one first, and its place there. */
const comparePlacement = (one: AnyTabKind, other: AnyTabKind): number => {
  const rank = (kind: AnyTabKind) => (kind.placement?.pane === 'main' ? 0 : 1)
  return rank(one) - rank(other) || (one.placement?.order ?? 0) - (other.placement?.order ?? 0)
}

/** What is told of a placed kind whose tab failed to open. */
type PlacementError = (kind: string, error: unknown) => void

/** A tab of every kind that declares a placement, laid out as each declares. */
async function createPlacedWorkspace(
  kinds: readonly AnyTabKind[],
  createTab: (kind: string) => Promise<string>,
  onError: PlacementError,
): Promise<Workspace> {
  const main: string[] = []
  const aside: string[] = []
  for (const one of kinds.filter((kind) => kind.placement).sort(comparePlacement)) {
    try {
      const id = await createTab(one.kind)
      if (one.placement?.pane === 'main') main.push(id)
      else aside.push(id)
    } catch (error) {
      onError(one.kind, error)
    }
  }
  return createWorkspace(main, aside)
}

/** What a tab holds, made in its scope; a tab that fails to open lets go of whatever it started. */
const openInScope = <T>(scope: EffectScope, open: () => T): T | undefined => {
  try {
    return scope.run(open)
  } catch (error) {
    scope.stop()
    throw error
  }
}

/** Effect scope manager for active tabs. */
function createTabScopes() {
  const scopes = new Map<string, EffectScope>()
  const drop = (id: string) => {
    scopes.get(id)?.stop()
    // Stryker disable next-line CallExpression: a stopped scope left in the map is stopped again as a no-op
    scopes.delete(id)
  }
  return { scopes, drop }
}

interface TabStoreContext {
  open: ShallowRef<ReadonlyMap<string, WindowTab>>
  layout: Ref<Workspace>
  drop: (id: string) => void
}

interface TabOpenerContext extends TabStoreContext {
  byKind: Map<string, AnyTabKind>
  scopes: Map<string, EffectScope>
}

/** Queries for inspecting tabs in the window. */
function createTabQueries(
  open: ShallowRef<ReadonlyMap<string, WindowTab>>,
  layout: Ref<Workspace>,
) {
  const getTab = (id: string): WindowTab | null => open.value.get(id) ?? null

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

/** Operations for creating and opening tabs. */
function createTabOpeners(ctx: TabOpenerContext) {
  /** A tab the core kept, opened under its id with the state its kind saved. */
  const openSavedTab = (saved: SavedTab) => {
    const one = ctx.byKind.get(saved.kind) ?? createUnavailableKind(saved.kind)
    const scope = effectScope(true)
    const state = openInScope(scope, () => one.open(saved.path, saved.id, saved.saved))
    ctx.scopes.set(saved.id, scope)
    ctx.open.value = new Map(ctx.open.value).set(saved.id, { kind: one, state })
  }

  const createTabSync = (kind: string, targetPath: string): string => {
    const one = ctx.byKind.get(kind) ?? createUnavailableKind(kind)
    const id = one.identity ? `${kind}:${one.identity(targetPath)}` : generateId(kind)
    if (ctx.open.value.has(id)) return id
    const scope = effectScope(true)
    const state = openInScope(scope, () => one.open(targetPath, id))
    ctx.scopes.set(id, scope)
    ctx.open.value = new Map(ctx.open.value).set(id, { kind: one, state })
    return id
  }

  const createTab = async (kind: string, targetPath = ''): Promise<string> =>
    createTabSync(kind, targetPath)

  const show = (id: string) => {
    ctx.layout.value = openTab(ctx.layout.value, id)
  }

  const openTabOfKind = async (kind: string, target = ''): Promise<string> => {
    const id = createTabSync(kind, target)
    show(id)
    return id
  }

  const openTabBesideOfKind = async (kind: string, target = ''): Promise<string> => {
    const id = createTabSync(kind, target)
    ctx.layout.value = openTabBeside(ctx.layout.value, id, {
      side: 'right',
      createId: generatePaneId,
    })
    return id
  }

  const finishClose = (id: string) => {
    ctx.open.value = omitEntry(ctx.open.value, id)
    ctx.drop(id)
    ctx.layout.value = closeTab(ctx.layout.value, id)
  }

  const onTabShown = (id: string) => {
    const one = ctx.open.value.get(id)
    if (!one) return
    ctx.open.value = new Map([...omitEntry(ctx.open.value, id), [id, one]])
    one.kind.onShow?.(one.state, id)
  }

  return {
    createTab,
    openSavedTab,
    show,
    openTabOfKind,
    openTabBesideOfKind,
    finishClose,
    onTabShown,
  }
}

/** Operations for releasing and closing tabs. */
function createTabClosers(ctx: TabStoreContext) {
  const releaseTab = (id: string): boolean => {
    const one = ctx.open.value.get(id)
    if (!one) return true
    if (one.kind.onClose && !one.kind.onClose(one.state, id)) return false
    ctx.open.value = omitEntry(ctx.open.value, id)
    ctx.drop(id)
    return true
  }

  const requestClose = (id: string) => {
    if (releaseTab(id)) ctx.layout.value = closeTab(ctx.layout.value, id)
  }

  const reconcileTab = (path: string, newKind: string) => {
    if (!path) return
    for (const [id, one] of ctx.open.value) {
      if (one.kind.kind === newKind || one.kind.isTool) continue
      if (getTabPath(one) === path) requestClose(id)
    }
  }

  const close = () => {
    for (const [id, one] of ctx.open.value) {
      if (one.kind.onDestroy) one.kind.onDestroy(one.state, id)
      else one.kind.onClose?.(one.state, id)
      ctx.drop(id)
    }
  }

  return { releaseTab, requestClose, reconcileTab, close }
}

/** Moving the tabs of a kind that navigates to another path. */
function createTabNavigation(
  open: ShallowRef<ReadonlyMap<string, WindowTab>>,
  openers: Pick<ReturnType<typeof createTabOpeners>, 'openTabOfKind' | 'show'>,
) {
  /**
   * The tab of a kind the person was last in, put in front and moved to the
   * path; a tab of it opened there where the window holds none.
   */
  const navigateTab = async (kind: string, targetPath: string): Promise<string> => {
    const last = [...open.value].filter(([, one]) => one.kind.kind === kind).at(-1)
    if (!last) return openers.openTabOfKind(kind, targetPath)
    const [id, one] = last
    openers.show(id)
    await one.kind.navigate?.(one.state, targetPath, '')
    return id
  }

  /** Every tab standing on a path, of a kind that navigates, moved to another. */
  const navigateTabsAt = async (from: string, to: string): Promise<void> => {
    const matchingTabs = [...open.value.values()].filter(
      (one) => one.kind.navigate && getTabPath(one) === from,
    )
    await Promise.all(
      matchingTabs.map((one) => Promise.resolve(one.kind.navigate?.(one.state, to, ''))),
    )
  }

  /**
   * Text put in the tab of a kind the person was last in, and put in front; a
   * tab of the kind opened to take it where the window holds none.
   */
  const sendText = async (kind: string, text: string): Promise<string> => {
    const last = [...open.value].filter(([, one]) => one.kind.kind === kind).at(-1)
    const id = last?.[0] ?? (await openers.openTabOfKind(kind))
    const one = open.value.get(id)
    one?.kind.receiveText?.(one.state, text)
    openers.show(id)
    return id
  }

  return { navigateTab, navigateTabsAt, sendText }
}

interface WindowHandleDeps {
  openTabOfKind: (kind: string, target?: string) => Promise<string>
  openTabBesideOfKind: (kind: string, target?: string) => Promise<string>
  show: (id: string) => void
  finishClose: (id: string) => void
  each: <T>(kind: string) => readonly KindTab<T>[]
  front: () => ActiveTab | null
  getTabStateIn: <T>(id: string, kind: string) => T | null
}

function createWindowHandle(deps: WindowHandleDeps): WindowHandle {
  return {
    openTab: (kind, target) => deps.openTabOfKind(kind, target),
    openTabBeside: (kind, target) => deps.openTabBesideOfKind(kind, target),
    show: (id) => deps.show(id),
    closeTab: (id) => deps.finishClose(id),
    each: <T>(kind: string) => deps.each<T>(kind),
    last: <T>(kind: string) => deps.each<T>(kind).at(-1) ?? null,
    front: () => deps.front(),
    getTabState: <T>(kind: string, id: string) => deps.getTabStateIn<T>(id, kind),
  }
}

/** Dispatches a key press to the active or most recently used tab. */
function dispatchTabKeyPress(
  layout: Workspace,
  open: ReadonlyMap<string, WindowTab>,
  event: KeyboardEvent,
): boolean {
  const activeTabIds = getPanes(layout.root).map((one) => one.active)
  const front = paneById(layout.root, layout.focus)?.active
  const proposed = new Set<string>()
  for (const id of [front, ...[...open.keys()].reverse()]) {
    if (!id || proposed.has(id) || !activeTabIds.includes(id)) continue
    proposed.add(id)
    const one = open.get(id)
    if (one?.kind.onKeyPress?.(one.state, event)) return true
  }
  return false
}

/** One tab release, and the rest kept. */
const omitEntry = <T>(target: ReadonlyMap<string, T>, id: string): ReadonlyMap<string, T> => {
  const rest = new Map(target)
  rest.delete(id)
  return rest
}
