/**
 * The window as a set of kinds, and the tabs it keeps of each.
 *
 * A kind says how one of its tabs opens, what it is called, what is drawn in
 * it and what letting go of it comes to. Everything here is written once for
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
import { closeTab, openTab, openTabBeside, pane, paneById, getPanes } from '@numen/ui'
import type { Tab, Workspace } from '@numen/ui'
import type { ActiveTab, AnyTabKind, KindTab, WindowHandle, WindowTab } from '../lib/kinds'
import { generateId } from '../lib/workspace'
import { createUnavailableKind } from '../lib/unavailable'

/** The identity a pane made by a split is filed under. */
const generatePaneId = () => crypto.randomUUID()

export function useWindowTabs() {
  const { byKind, registerKinds, registerKind } = createTabRegistry()
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

  const queries = createTabQueries(open, layout)
  const openers = createTabOpeners({ open, layout, byKind, scopes, drop })
  const closers = createTabClosers({ open, layout, drop })
  const handle = createWindowHandle({ ...openers, ...queries })
  const onKeyPress = (event: KeyboardEvent): boolean =>
    dispatchTabKeyPress(layout.value, open.value, event)

  return {
    layout,
    tabs,
    handle,
    registerKinds,
    registerKind,
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
}

/** Registry of tab kinds supported by the window. */
function createTabRegistry() {
  const byKind = new Map<string, AnyTabKind>()
  const registerKinds = (kinds: readonly AnyTabKind[]) => {
    for (const one of kinds) byKind.set(one.kind, one)
  }
  const registerKind = (kind: AnyTabKind): (() => void) => {
    byKind.set(kind.kind, kind)
    return () => {
      if (byKind.get(kind.kind) === kind) byKind.delete(kind.kind)
    }
  }
  return { byKind, registerKinds, registerKind }
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
  const createTab = async (kind: string, at = ''): Promise<string> => {
    const one = ctx.byKind.get(kind) ?? createUnavailableKind(kind)
    const id = one.identity ? `${kind}:${one.identity(at)}` : generateId(kind)
    if (ctx.open.value.has(id)) return id
    const scope = effectScope(true)
    const state = scope.run(() => one.open(at))
    ctx.scopes.set(id, scope)
    ctx.open.value = new Map(ctx.open.value).set(id, { kind: one, state })
    return id
  }

  const show = (id: string) => {
    ctx.layout.value = openTab(ctx.layout.value, id)
  }

  const openTabOfKind = async (kind: string, at = ''): Promise<string> => {
    const id = await createTab(kind, at)
    if (id) show(id)
    return id
  }

  const openTabBesideOfKind = async (kind: string, at = ''): Promise<string> => {
    const id = await createTab(kind, at)
    if (id) {
      ctx.layout.value = openTabBeside(ctx.layout.value, id, {
        side: 'right',
        createId: generatePaneId,
      })
    }
    return id
  }

  const finishClose = (id: string) => {
    ctx.open.value = without(ctx.open.value, id)
    ctx.drop(id)
    ctx.layout.value = closeTab(ctx.layout.value, id)
  }

  const onTabShown = (id: string) => {
    const one = ctx.open.value.get(id)
    if (!one) return
    ctx.open.value = new Map([...without(ctx.open.value, id), [id, one]])
    one.kind.onShow?.(one.state, id)
  }

  return { createTab, show, openTabOfKind, openTabBesideOfKind, finishClose, onTabShown }
}

/** Operations for releasing and closing tabs. */
function createTabClosers(ctx: TabStoreContext) {
  const releaseTab = (id: string): boolean => {
    const one = ctx.open.value.get(id)
    if (!one) return true
    if (one.kind.onClose && !one.kind.onClose(one.state, id)) return false
    ctx.open.value = without(ctx.open.value, id)
    ctx.drop(id)
    return true
  }

  const requestClose = (id: string) => {
    if (releaseTab(id)) ctx.layout.value = closeTab(ctx.layout.value, id)
  }

  const reconcileTab = (path: string, newKind: string) => {
    if (!path) return
    for (const [id, one] of ctx.open.value) {
      if (one.kind.kind === newKind || TOOL_KINDS.has(one.kind.kind)) continue
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

interface WindowHandleDeps {
  openTabOfKind: (kind: string, at?: string) => Promise<string>
  openTabBesideOfKind: (kind: string, at?: string) => Promise<string>
  show: (id: string) => void
  finishClose: (id: string) => void
  each: <T>(kind: string) => readonly KindTab<T>[]
  front: () => ActiveTab | null
  getTabStateIn: <T>(id: string, kind: string) => T | null
}

function createWindowHandle(deps: WindowHandleDeps): WindowHandle {
  return {
    openTab: (kind, at) => deps.openTabOfKind(kind, at),
    openTabBeside: (kind, at) => deps.openTabBesideOfKind(kind, at),
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
  const drawn = getPanes(layout.root).map((one) => one.active)
  const front = paneById(layout.root, layout.focus)?.active
  const offered = new Set<string>()
  for (const id of [front, ...[...open.keys()].reverse()]) {
    if (!id || offered.has(id) || !drawn.includes(id)) continue
    offered.add(id)
    const one = open.get(id)
    if (one?.kind.onKeyPress?.(one.state, event)) return true
  }
  return false
}

/** Non-file tool tabs that are preserved across file tab reconciliations. */
const TOOL_KINDS = new Set(['agent', 'files', 'plex', 'settings'])

/** The file path a tab stands on. */
const getTabPath = (tab: WindowTab): string => {
  const openTab = tab.kind.getOpenTab?.(tab.state)
  if (openTab?.path) return openTab.path
  const target = tab.kind.getTarget?.(tab.state)
  // Stryker disable next-line StringLiteral: reconcileTab returns before comparing when the path asked for is empty, so the empty fallback is never compared
  return target?.path || target?.file || ''
}

/** One tab let go of, and the rest kept. */
const without = <T>(held: ReadonlyMap<string, T>, id: string): ReadonlyMap<string, T> => {
  const rest = new Map(held)
  rest.delete(id)
  return rest
}
