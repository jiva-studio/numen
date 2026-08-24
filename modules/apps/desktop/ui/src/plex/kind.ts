/**
 * What one plex tab holds: where it is standing, and what a gesture in it does.
 *
 * The plex reports the shape of a gesture and nothing else. What making a note
 * from a node comes to, what the menu on a node offers, and which note a click
 * opens are decided here, so a test can ask them without a screen.
 */
import { computed, ref } from 'vue'
import type { MenuOpening, PlexNeighbourhood, PlexRelatedSeat, PlexShowing } from '@numen/ui'
import { chose as carry } from '../menu'
import { asPlex } from '../plex'
import type { Standing } from '../standing'
import type { Host, Kind } from '../windowing'
import { WORDS as words } from '../words'
import { PLEX, plexCalled } from '../workspace'
import PlexTab from './PlexTab.vue'

/** Where the menu on a node stands, and what it was asked for on. */
export interface Asked {
  readonly path: string
  readonly at: { x: number; y: number }
  readonly from: HTMLElement | SVGElement | null
  readonly opening: MenuOpening
}

/** Making a note from the picture, and joining two that are already on it. */
export interface Making {
  make(from: string, seat: PlexRelatedSeat): Promise<unknown>
  join(from: string, to: string, seat: PlexRelatedSeat): Promise<boolean>
}

/** What a plex tab asks of the vault and of the window it is drawn in. */
export interface Plexing {
  readonly makes: Making
  /** Whether the window has anything true to draw at all. */
  ready(): boolean
  /** A note opened in a tab of its own, under the name the picture gives it. */
  opens(path: string, title: string, showing: PlexShowing): void
  /** Something to ask, put in the agent the person was last in. */
  asks(text: string): void
  /** The note the vault opens with, where a plex has nowhere else to stand. */
  opening(): string
}

/** What one plex tab holds. */
export type Held = ReturnType<typeof plexing>

/**
 * The plex tabs of a window, in the order the person was last in them.
 *
 * The plex the person is looking at is the one a note asked for from outside
 * the window is put in front of, the one whose trouble the window says, and
 * the one a plex opened after it stands beside.
 */
export function plexKind(host: Host, makes: () => Standing, deps: Plexing) {
  const open = new Map<string, Held>()
  /** The identities of the tabs, the last of them in front. */
  let order: readonly string[] = []

  /** What the plex in front holds, and nothing while the window holds none. */
  const front = (): Held | null => open.get(order.at(-1) ?? '') ?? null

  const kind: Kind<Held> = {
    kind: PLEX,
    opens: (at, id) => {
      const held = plexing(makes(), deps)
      const from = at || looking() || deps.opening()
      open.set(id, held)
      order = [...order, id]
      if (from) void held.view.go(from)
      return held
    },
    called: (held) => plexCalled(words.plex, held.view.neighbourhood.value?.focus?.title ?? ''),
    draws: PlexTab,
    shown: (_held, id) => {
      if (open.has(id)) order = [...order.filter((one) => one !== id), id]
    },
    shuts: (held, id) => {
      held.view.close()
      open.delete(id)
      order = order.filter((one) => one !== id)
      return true
    },
    offers: words.newPlex,
  }

  /** The note the person is looking at, which is what a question is about. */
  const looking = (): string => front()?.view.here.value ?? ''

  /** What the plex in front could not show, for the window to put up. */
  const trouble = (): string => front()?.view.trouble.value ?? ''

  /**
   * A note put in front of the person: the plex they are looking at travels
   * there, and a window holding no plex at all opens one on it.
   */
  const travel = async (path: string) => {
    const one = front()
    if (one) await one.view.go(path)
    else await host.opens(PLEX, path)
  }

  /**
   * Every plex asks for its picture again. One standing nowhere is given the
   * note the vault opens with.
   */
  const again = async () => {
    await Promise.all(
      [...open.values()].map((one) => {
        const path = one.view.here.value || deps.opening()
        return path ? one.view.go(path) : Promise.resolve()
      }),
    )
  }

  /** Whether any plex is standing nowhere, and so wants a note to stand on. */
  const nowhere = (): boolean => [...open.values()].some((one) => !one.view.here.value)

  return { kind, looking, trouble, travel, again, nowhere }
}

export function plexing(view: Standing, deps: Plexing) {
  /** The picture as the plex reads it, and nothing while the window has none. */
  const picture = computed<PlexNeighbourhood | null>(() =>
    deps.ready() && view.neighbourhood.value ? asPlex(view.neighbourhood.value) : null,
  )

  /** The menu on a node, for as long as it stands. */
  const menu = ref<Asked | null>(null)

  /** A node chosen: the plex travels there, and the picture is asked for again. */
  const activate = (path: string) => void view.go(path)

  /**
   * A note made in a seat of another one. It is in the index by the time the
   * answer arrives, so the picture is asked for again and it is drawn in it.
   *
   * The plex stands on the note it was made from: a neighbourhood is one seat
   * deep, and that is the seat the new note sits in.
   */
  const made = async (from: string, seat: PlexRelatedSeat) => {
    if (await deps.makes.make(from, seat)) await view.go(from)
  }

  /**
   * Two notes the person drew a line between. The plex stands on the note the
   * line was drawn from, which is the one the link is written in.
   */
  const joined = async (from: string, to: string, seat: PlexRelatedSeat) => {
    if (await deps.makes.join(from, to, seat)) await view.go(from)
  }

  /** A note opened where the person asked for it, called what the picture calls it. */
  const opens = (path: string, showing: PlexShowing = 'here') =>
    deps.opens(path, nameOf(path), showing)

  /** A menu asked for on a node, and one put away. */
  const asks = (asked: Asked) => {
    menu.value = asked
  }
  const dismiss = () => {
    menu.value = null
  }

  /** An item chosen in the menu, on the note it was asked for on. */
  const chose = (id: string) => {
    const asking = menu.value
    menu.value = null
    if (!asking) return
    carry(id, asking.path, {
      open: (path) => opens(path),
      child: (path) => void made(path, 'child'),
      ask: (path) => deps.asks(`${path} — `),
      copy: (path) => void navigator.clipboard?.writeText(path),
    })
  }

  /**
   * What a note is called, as the picture has it. A note the picture does not
   * name is called by the file it is filed under.
   */
  const nameOf = (path: string): string => {
    const around = view.neighbourhood.value
    if (around?.focus?.path === path && around.focus.title) return around.focus.title
    const near = around?.related?.find((one) => one.note?.path === path)
    return near?.note?.title || (path.split('/').pop() ?? path).replace(/\.md$/, '')
  }

  return { view, picture, menu, activate, made, joined, opens, asks, dismiss, chose, nameOf }
}
