/**
 * The window as a set of kinds, and the tabs it keeps of each.
 *
 * A kind says how one of its tabs opens, what it is called, what is drawn in
 * it and what letting go of it comes to. Everything here is written once for
 * all of them: a kind is added by declaring one, and the window is not told
 * about it twice.
 */
import { computed, effectScope, ref, shallowRef, type EffectScope, type Ref } from 'vue'
import { closeTab, openTab, openTabBeside, pane, paneById, panesOf } from '@numen/ui'
import type { Tab, Workspace } from '@numen/ui'
import type { ActiveTab, AnyTabKind, KindTab, WindowHandle, WindowTab } from './kinds'
import { generateId } from './workspace'

/** The identity a pane made by a split is filed under. */
const generatePaneId = () => crypto.randomUUID()

export function useWindowTabs() {
  /**
   * What every kind is given. It is there before any kind is, so a kind is
   * made with it and declared to the window it already has.
   */
  const handle: WindowHandle = {
    opens: (kind, at) => opens(kind, at),
    beside: (kind, at) => beside(kind, at),
    show: (id) => show(id),
    closes: (id) => finishClose(id),
    each: <TabState,>(kind: string) => each<TabState>(kind),
    last: <TabState,>(kind: string) => each<TabState>(kind).at(-1) ?? null,
    front: () => front(),
    holds: <TabState,>(kind: string, id: string) => holdsIn<TabState>(id, kind),
  }

  /** The kinds of tab this window draws, each under the word it is asked for by. */
  const byKind = new Map<string, AnyTabKind>()

  /**
   * The kinds this window draws. It is told once, before a tab of any of them
   * is opened.
   */
  const registerKinds = (told: readonly AnyTabKind[]) => {
    for (const one of told) byKind.set(one.kind, one)
  }

  /**
   * Every tab the window holds, each under the identity it opened with, in the
   * order the person was last in them.
   */
  const open = shallowRef<ReadonlyMap<string, WindowTab>>(new Map())

  /**
   * What each open tab watches. A kind makes what its tab holds outside any
   * component, and the scope is what stops it when the tab goes.
   */
  const scopes = new Map<string, EffectScope>()

  /** A tab is gone, and nothing it started keeps answering. */
  const drop = (id: string) => {
    scopes.get(id)?.stop()
    scopes.delete(id)
  }

  const layout: Ref<Workspace> = ref({
    root: pane('main', []),
    axis: 'horizontal',
    focus: 'main',
  })

  /** What each tab of the window is called, and the word it carries. */
  const tabs = computed<readonly Tab[]>(() =>
    [...open.value].map(([id, one]): Tab => {
      const mark = one.kind.marked?.(one.state)
      const title = (one.kind.getTitle ?? one.kind.called)?.(one.state) ?? ''
      return { id, title, ...(mark ? { mark } : {}) }
    }),
  )

  /** What one tab holds, or nothing where the window holds no such tab. */
  const getTab = (id: string): WindowTab | null => open.value.get(id) ?? null

  /**
   * What one tab of a kind holds, for a caller that knows the kind and what
   * its tabs hold. A tab of another kind is nothing to it.
   */
  const holdsIn = <T,>(id: string, kind: string): T | null => {
    const one = open.value.get(id)
    return one && one.kind.kind === kind ? (one.state as T) : null
  }

  /**
   * The tab the person is looking at, which is the one showing in the pane the
   * layout is focused on. A tab the window holds answers under its kind, and
   * one it does not hold answers under none.
   */
  const front = (): ActiveTab | null => {
    const id = paneById(layout.value.root, layout.value.focus)?.active
    if (!id) return null
    const one = open.value.get(id)
    return one ? { id, kind: one.kind.kind, state: one.state } : { id, kind: null, state: null }
  }

  /** Every tab of a kind, the one the person was last in last. */
  const each = <T,>(kind: string): readonly KindTab<T>[] =>
    [...open.value]
      .filter(([, one]) => one.kind.kind === kind)
      .map(([id, one]) => ({ id, state: one.state as T }))

  /**
   * A tab of a kind, on what it was given. A kind that takes its identity from
   * that answers with the tab it already has.
   *
   * What a kind calls one of its tabs is filed under that kind, so two kinds
   * that name a tab after the same thing hold a tab each.
   */
  const createTab = async (kind: string, at = ''): Promise<string> => {
    const one = byKind.get(kind)
    if (!one) return ''
    const id = one.identity ? `${kind}:${one.identity(at)}` : generateId(kind)
    if (open.value.has(id)) return id
    const scope = effectScope(true)
    const state = scope.run(() => one.opens(at))
    scopes.set(id, scope)
    open.value = new Map(open.value).set(id, { kind: one, state })
    return id
  }
  const makes = createTab

  /** A tab opened where the person is, and put in front. */
  const opens = async (kind: string, at = ''): Promise<string> => {
    const id = await createTab(kind, at)
    if (id) show(id)
    return id
  }

  /** A tab opened beside the pane the person is in. */
  const beside = async (kind: string, at = ''): Promise<string> => {
    const id = await createTab(kind, at)
    if (id) layout.value = openTabBeside(layout.value, id, 'right', generatePaneId)
    return id
  }

  /** A tab the window already holds, put in front. */
  const show = (id: string) => {
    layout.value = openTab(layout.value, id)
  }

  /** A tab that took its own close, going now. */
  const finishClose = (id: string) => {
    open.value = without(open.value, id)
    drop(id)
    layout.value = closeTab(layout.value, id)
  }

  /**
   * The tab now on screen, where what it holds has room to measure. It goes to
   * the end of what the window holds, which is the order they were last in.
   */
  const onTabShown = (id: string) => {
    const one = open.value.get(id)
    if (!one) return
    open.value = new Map([...without(open.value, id), [id, one]])
    ;(one.kind.onShow ?? one.kind.shown)?.(one.state, id)
  }

  /**
   * A key struck, offered to the tabs on screen: the one in the pane the person
   * is in first, and then the rest, the one they were in last before the one
   * before that. The first to take it keeps it.
   *
   * A pane is what the layout is focused on and not what the person is looking
   * at, and several panes are drawn at once. A key nobody in the pane it names
   * reads is one the tab in front of them would have.
   */
  const onKeyPress = (event: KeyboardEvent): boolean => {
    const drawn = panesOf(layout.value.root).map((one) => one.active)
    const front = paneById(layout.value.root, layout.value.focus)?.active
    const offered = new Set<string>()
    for (const id of [front, ...[...open.value.keys()].reverse()]) {
      if (!id || offered.has(id) || !drawn.includes(id)) continue
      offered.add(id)
      const one = open.value.get(id)
      const handler = one?.kind.onKeyPress ?? one?.kind.presses
      if (handler?.(one!.state, event)) return true
    }
    return false
  }

  /**
   * A tab lets go of what it held, and says whether it went. A kind that has
   * something to finish keeps the tab and closes it itself.
   */
  const shut = (id: string): boolean => {
    const one = open.value.get(id)
    if (!one) return true
    const canClose = one.kind.onClose ?? one.kind.shuts
    if (canClose && !canClose(one.state, id)) return false
    open.value = without(open.value, id)
    drop(id)
    return true
  }

  /**
   * A tab let go of from outside and taken off the screen. A kind with
   * something to finish keeps its tab and closes it itself.
   */
  const requestClose = (id: string) => {
    if (shut(id)) layout.value = closeTab(layout.value, id)
  }

  /** The window is going, and nothing a tab holds outlives it. */
  const close = () => {
    for (const [id, one] of open.value) {
      const onDestroy = one.kind.onDestroy ?? one.kind.gone
      if (onDestroy) onDestroy(one.state, id)
      else (one.kind.onClose ?? one.kind.shuts)?.(one.state, id)
      drop(id)
    }
  }

  return {
    layout,
    tabs,
    handle,
    registerKinds,
    getTab,
    holdsIn,
    createTab,
    makes,
    opens,
    beside,
    show,
    finishClose,
    onTabShown,
    onKeyPress,
    shut,
    requestClose,
    close,
  }
}

/** One tab let go of, and the rest kept. */
const without = <T,>(held: ReadonlyMap<string, T>, id: string): ReadonlyMap<string, T> => {
  const rest = new Map(held)
  rest.delete(id)
  return rest
}
