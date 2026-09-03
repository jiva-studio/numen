/**
 * The window as a set of kinds, and the tabs it keeps of each.
 *
 * A kind says how one of its tabs opens, what it is called, what is drawn in
 * it and what letting go of it comes to. Everything here is written once for
 * all of them: a kind is added by declaring one, and the window is not told
 * about it twice.
 */
import { computed, ref, shallowRef, type Component, type Ref } from 'vue'
import { closeTab, openTab, openTabBeside, pane, paneById } from '@numen/ui'
import type { Tab, WorkspaceLayout } from '@numen/ui'
import type { Source } from './core'
import { named } from './workspace'

/**
 * What a command asked over one tab is over: the note the tab means, and the
 * file a run is asked over. A kind that means neither answers with neither.
 */
export interface At {
  readonly path?: string
  readonly title?: string
  readonly file?: string
  readonly source?: Source | null
}

/**
 * What one tab holds, as whoever answers on the person's behalf is told it: the
 * file it stands at, where in it the person is, and how much of it there is.
 */
export interface Attends {
  readonly path: string
  readonly at?: number
  readonly of?: number
}

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
  shown?(held: Held, id: string): void
  /** What a command asked over one of its tabs is over. */
  at?(held: Held): At
  /** What one of its tabs holds, as whoever answers for the person is told it. */
  attends?(held: Held): Attends
  /**
   * The tab lets go of what it held. False keeps it on screen: what it holds
   * has something to finish, and closes the tab itself once it has.
   */
  shuts?(held: Held, id: string): boolean
  /**
   * The window is going, and nothing this tab holds outlives it. A kind that
   * says nothing here lets go the way a tab of it closes.
   */
  gone?(held: Held, id: string): void
}

/**
 * A kind as the window keeps it. What its tabs hold is the kind's own affair,
 * and the window hands it back to the kind untouched.
 */
export type Kept = Kind<unknown>

/** What one tab is: its kind, and what that kind gave it to hold. */
export interface Open {
  readonly kind: Kept
  readonly held: unknown
}

/** One tab of a kind, as that kind is given it back. */
export interface Tabbed<Held> {
  readonly id: string
  readonly held: Held
}

/** The tab the person is looking at, whichever kind it turns out to be. */
export interface Fronted {
  readonly id: string
  /** The word its kind is filed under, and nothing where the window holds no such tab. */
  readonly kind: string | null
  /** What its kind gave it to hold, and nothing where the window holds no such tab. */
  readonly held: unknown
}

/** What a kind may ask of the window its tabs are drawn in. */
export interface Host {
  /** A tab of a kind, opened on something and put in front. */
  opens(kind: string, at?: string): Promise<string>
  /** The same, drawn beside the pane the person is in. */
  beside(kind: string, at?: string): Promise<string>
  /** A tab the window already holds, put in front. */
  shows(id: string): void
  /** A tab that took its own close, going now. */
  closes(id: string): void
  /**
   * Every tab of a kind, in the order the person was last in them. The last of
   * them is the one in front.
   */
  each<Held>(kind: string): readonly Tabbed<Held>[]
  /** The tab of a kind the person was last in, and nothing where it holds none. */
  last<Held>(kind: string): Tabbed<Held> | null
  /**
   * The tab showing in the pane the person is in, of whatever kind. A pane
   * holding nothing answers with nothing.
   */
  front(): Fronted | null
  /** What one tab of a kind holds, and nothing where the tab is another kind. */
  holds<Held>(kind: string, id: string): Held | null
}

/** The identity a pane made by a split is filed under. */
const naming = () => crypto.randomUUID()

export function windowing() {
  /**
   * What every kind is given. It is there before any kind is, so a kind is
   * made with it and declared to the window it already has.
   */
  const host: Host = {
    opens: (kind, at) => opens(kind, at),
    beside: (kind, at) => beside(kind, at),
    shows: (id) => shows(id),
    closes: (id) => closes(id),
    each: <Held,>(kind: string) => each<Held>(kind),
    last: <Held,>(kind: string) => each<Held>(kind).at(-1) ?? null,
    front: () => front(),
    holds: <Held,>(kind: string, id: string) => holdsIn<Held>(id, kind),
  }

  /** The kinds of tab this window draws, each under the word it is asked for by. */
  const byKind = new Map<string, Kept>()

  /**
   * The kinds this window draws. It is told once, before a tab of any of them
   * is opened.
   */
  const declares = (told: readonly Kept[]) => {
    for (const one of told) byKind.set(one.kind, one)
  }

  /**
   * Every tab the window holds, each under the identity it opened with, in the
   * order the person was last in them.
   */
  const open = shallowRef<ReadonlyMap<string, Open>>(new Map())
  const layout: Ref<WorkspaceLayout> = ref({
    root: pane('main', []),
    axis: 'horizontal',
    focus: 'main',
  })

  /** What each tab of the window is called, and the word it carries. */
  const tabs = computed<readonly Tab[]>(() =>
    [...open.value].map(([id, one]): Tab => {
      const mark = one.kind.marked?.(one.held)
      return { id, title: one.kind.called(one.held), ...(mark ? { mark } : {}) }
    }),
  )

  /** What one tab holds, or nothing where the window holds no such tab. */
  const heldIn = (id: string): Open | null => open.value.get(id) ?? null

  /**
   * What one tab of a kind holds, for a caller that knows the kind and what
   * its tabs hold. A tab of another kind is nothing to it.
   */
  const holdsIn = <T,>(id: string, kind: string): T | null => {
    const one = open.value.get(id)
    return one && one.kind.kind === kind ? (one.held as T) : null
  }

  /**
   * The tab the person is looking at, which is the one showing in the pane the
   * layout is focused on. A tab the window holds answers under its kind, and
   * one it does not hold answers under none.
   */
  const front = (): Fronted | null => {
    const id = paneById(layout.value.root, layout.value.focus)?.active
    if (!id) return null
    const one = open.value.get(id)
    return one ? { id, kind: one.kind.kind, held: one.held } : { id, kind: null, held: null }
  }

  /** Every tab of a kind, the one the person was last in last. */
  const each = <T,>(kind: string): readonly Tabbed<T>[] =>
    [...open.value]
      .filter(([, one]) => one.kind.kind === kind)
      .map(([id, one]) => ({ id, held: one.held as T }))

  /**
   * A tab of a kind, on what it was given. A kind that takes its identity from
   * that answers with the tab it already has.
   *
   * What a kind calls one of its tabs is filed under that kind, so two kinds
   * that name a tab after the same thing hold a tab each.
   */
  const makes = async (kind: string, at = ''): Promise<string> => {
    const one = byKind.get(kind)
    if (!one) return ''
    const id = one.identity ? `${kind}:${one.identity(at)}` : named(kind)
    if (open.value.has(id)) return id
    const held = await one.opens(at)
    open.value = new Map(open.value).set(id, { kind: one, held })
    return id
  }

  /** A tab opened where the person is, and put in front. */
  const opens = async (kind: string, at = ''): Promise<string> => {
    const id = await makes(kind, at)
    if (id) shows(id)
    return id
  }

  /** A tab opened beside the pane the person is in. */
  const beside = async (kind: string, at = ''): Promise<string> => {
    const id = await makes(kind, at)
    if (id) layout.value = openTabBeside(layout.value, id, 'right', naming)
    return id
  }

  /** A tab the window already holds, put in front. */
  const shows = (id: string) => {
    layout.value = openTab(layout.value, id)
  }

  /** A tab that took its own close, going now. */
  const closes = (id: string) => {
    open.value = without(open.value, id)
    layout.value = closeTab(layout.value, id)
  }

  /**
   * The tab now on screen, where what it holds has room to measure. It goes to
   * the end of what the window holds, which is the order they were last in.
   */
  const shown = (id: string) => {
    const one = open.value.get(id)
    if (!one) return
    open.value = new Map([...without(open.value, id), [id, one]])
    one.kind.shown?.(one.held, id)
  }

  /**
   * A tab lets go of what it held, and says whether it went. A kind that has
   * something to finish keeps the tab and closes it itself.
   */
  const shut = (id: string): boolean => {
    const one = open.value.get(id)
    if (!one) return true
    if (one.kind.shuts && !one.kind.shuts(one.held, id)) return false
    open.value = without(open.value, id)
    return true
  }

  /**
   * A tab let go of from outside and taken off the screen. A kind with
   * something to finish keeps its tab and closes it itself.
   */
  const drops = (id: string) => {
    if (shut(id)) layout.value = closeTab(layout.value, id)
  }

  /** The window is going, and nothing a tab holds outlives it. */
  const close = () => {
    for (const [id, one] of open.value) {
      if (one.kind.gone) one.kind.gone(one.held, id)
      else one.kind.shuts?.(one.held, id)
    }
  }

  return {
    layout,
    tabs,
    host,
    declares,
    heldIn,
    holdsIn,
    opens,
    beside,
    shows,
    closes,
    shown,
    shut,
    drops,
    close,
  }
}

/** One tab let go of, and the rest kept. */
const without = <T,>(held: ReadonlyMap<string, T>, id: string): ReadonlyMap<string, T> => {
  const rest = new Map(held)
  rest.delete(id)
  return rest
}
