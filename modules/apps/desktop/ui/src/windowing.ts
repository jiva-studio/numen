/**
 * The window as a set of kinds, and the tabs it keeps of each.
 *
 * A kind says how one of its tabs opens, what it is called, what is drawn in
 * it and what letting go of it comes to. Everything here is written once for
 * all of them: a kind is added by declaring one, and the window is not told
 * about it twice.
 */
import { computed, ref, shallowRef, type Component, type Ref } from 'vue'
import { closeTab, openTab, openTabBeside, pane, paneWithTab } from '@numen/ui'
import type { NodeId, Tab, WorkspaceLayout } from '@numen/ui'
import { BLANK, named } from './workspace'

/** A kind of tab: what it holds, what it is called, and what it lets go of. */
export interface Kind<Held> {
  /** The word the identities of its tabs are filed under. */
  readonly kind: string
  /** What one of its tabs holds, made as the tab opens on what it was given. */
  opens(at: string): Held | Promise<Held>
  /** What the tab is called, as what it holds now stands. */
  called(held: Held): string
  /** The one word the tab carries beside its title, or nothing. */
  marked?(held: Held): string | undefined
  /** What is drawn in the pane, given what the tab holds. */
  readonly draws: Component
  /**
   * The identity a tab of this kind takes from what it opens on. A kind that
   * declares one has a tab per thing, so the same thing opened twice is the
   * tab it already has.
   */
  identity?(at: string): string
  /** The tab came on screen, where what it holds has room to measure. */
  shown?(held: Held): void
  /**
   * The tab lets go of what it held. False keeps it on screen: what it holds
   * has something to finish, and closes the tab itself once it has. The window
   * going says this once and waits for nothing.
   */
  shuts?(held: Held): boolean
  /** What a blank tab offers to become, or nothing. */
  readonly offers?: string
}

/**
 * A kind as the window keeps it. What its tabs hold is the kind's own affair,
 * and the window hands it back to the kind untouched.
 */
export type Kept = Kind<any>

/** What one tab is: its kind, and what that kind gave it to hold. */
export interface Open {
  readonly kind: Kept
  readonly held: unknown
}

/** What a kind may ask of the window its tabs are drawn in. */
export interface Host {
  /** A tab of a kind, opened on something and put in front. */
  opens(kind: string, at?: string): Promise<string>
  /** The same, drawn beside the pane the person is in. */
  beside(kind: string, at?: string): Promise<string>
  /** A tab that took its own close, going now. */
  closes(id: string): void
}

/** What the window itself says about its tabs. */
export interface Words {
  /** What a tab with nothing in it yet is called. */
  readonly newTab: string
}

/** The identity a pane made by a split is filed under. */
const naming = () => crypto.randomUUID()

export function windowing(kinds: readonly Kept[], words: Words) {
  const byKind = new Map(kinds.map((one) => [one.kind, one]))
  /** Every tab the window holds, each under the identity it opened with. */
  const open = shallowRef<ReadonlyMap<string, Open>>(new Map())
  /** Tabs opened with nothing in them, each waiting to be told what it holds. */
  const blanks = ref<readonly string[]>([])
  const layout: Ref<WorkspaceLayout> = ref({
    root: pane('main', []),
    axis: 'horizontal',
    focus: 'main',
  })

  /** What each tab of the window is called, and the word it carries. */
  const tabs = computed<readonly Tab[]>(() => [
    ...[...open.value].map(([id, one]): Tab => {
      const mark = one.kind.marked?.(one.held)
      return { id, title: one.kind.called(one.held), ...(mark ? { mark } : {}) }
    }),
    ...blanks.value.map((id): Tab => ({ id, title: words.newTab })),
  ])

  /** What one tab holds, or nothing where the window holds no such tab. */
  const heldIn = (id: string): Open | null => open.value.get(id) ?? null

  /**
   * A tab of a kind, on what it was given. A kind that takes its identity from
   * that answers with the tab it already has.
   */
  const makes = async (kind: string, at = ''): Promise<string> => {
    const one = byKind.get(kind)
    if (!one) return ''
    const id = one.identity ? one.identity(at) : named(kind)
    if (open.value.has(id)) return id
    const held = await one.opens(at)
    open.value = new Map(open.value).set(id, { kind: one, held })
    return id
  }

  /** A tab opened where the person is, and put in front. */
  const opens = async (kind: string, at = ''): Promise<string> => {
    const id = await makes(kind, at)
    if (id) layout.value = openTab(layout.value, id)
    return id
  }

  /** A tab opened beside the pane the person is in. */
  const beside = async (kind: string, at = ''): Promise<string> => {
    const id = await makes(kind, at)
    if (id) layout.value = openTabBeside(layout.value, id, 'right', naming)
    return id
  }

  /** A tab that took its own close, going now. */
  const closes = (id: string) => {
    open.value = without(open.value, id)
    layout.value = closeTab(layout.value, id)
  }

  const host: Host = { opens, beside, closes }

  /** A tab opened empty, in the pane the plus was pressed in. */
  const blanked = (pane: NodeId) => {
    const id = named(BLANK)
    blanks.value = [...blanks.value, id]
    layout.value = openTab(layout.value, id, pane)
  }

  /** A blank tab told what to hold, and holding it where it stood. */
  const becomeIt = async (blank: string, kind: string) => {
    const id = await makes(kind)
    if (!id) return
    const pane = paneWithTab(layout.value.root, blank)?.id
    layout.value = openTab(layout.value, id, pane ?? layout.value.focus)
    layout.value = closeTab(layout.value, blank)
    blanks.value = blanks.value.filter((one) => one !== blank)
  }

  /** What a blank tab offers to become, in the order the kinds were declared. */
  const becomes = computed<readonly Tab[]>(() =>
    kinds.filter((one) => one.offers).map((one) => ({ id: one.kind, title: one.offers ?? '' })),
  )

  /** The tab now on screen, where what it holds has room to measure. */
  const shown = (id: string) => {
    const one = open.value.get(id)
    one?.kind.shown?.(one.held)
  }

  /**
   * A tab lets go of what it held, and says whether it went. A kind that has
   * something to finish keeps the tab and closes it itself.
   */
  const shut = (id: string): boolean => {
    if (blanks.value.includes(id)) {
      blanks.value = blanks.value.filter((one) => one !== id)
      return true
    }
    const one = open.value.get(id)
    if (!one) return true
    if (one.kind.shuts && !one.kind.shuts(one.held)) return false
    open.value = without(open.value, id)
    return true
  }

  /** The window is going, and nothing a tab holds outlives it. */
  const close = () => {
    for (const one of open.value.values()) one.kind.shuts?.(one.held)
  }

  return {
    layout,
    tabs,
    blanks,
    becomes,
    host,
    heldIn,
    opens,
    beside,
    closes,
    blanked,
    becomeIt,
    shown,
    shut,
    close,
  }
}

/** One tab let go of, and the rest kept. */
const without = <T,>(held: ReadonlyMap<string, T>, id: string): ReadonlyMap<string, T> => {
  const rest = new Map(held)
  rest.delete(id)
  return rest
}
