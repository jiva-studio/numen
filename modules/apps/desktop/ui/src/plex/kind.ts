/**
 * What one plex tab holds: where it is standing, and what a gesture in it does.
 *
 * The plex reports the shape of a gesture and nothing else. What making a note
 * from a node comes to, what the menu on a node offers, and which note a click
 * opens are decided here, so a test can ask them without a screen.
 */
import { computed, ref } from 'vue'
import type { MenuOpening, PlexNeighbourhood, PlexRelatedSeat, PlexShowing } from '@numen/ui'
import { chose as carry } from './menu'
import { asPlex } from './picture'
import type { Standing } from './standing'
import type { Went } from '../core'
import type { Host, Kind } from '../windowing'
import { PLEX, plexCalled } from '../workspace'
import PlexTab from './PlexTab.vue'
import { WORDS as words } from './words'

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
  /**
   * A command asked for on a node, on the note it stands for. One that needs
   * something asks for it in the palette; the rest happen where they stand.
   */
  runs(id: string, path: string, title: string): void
  /** The note the vault opens with, as it was last answered. */
  opening(): string
  /** Asks the vault where it opens, for a plex that has nowhere to stand. */
  first(): Promise<string>
  /** The seats a gesture may make a note in, which the picture draws. */
  readonly creatable: readonly PlexRelatedSeat[]
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
  /** Every plex the window holds, and the one the person was last in. */
  const all = () => host.each<Held>(PLEX)
  const front = (): Held | null => host.last<Held>(PLEX)?.held ?? null

  const kind: Kind<Held> = {
    kind: PLEX,
    opens: (at) => {
      const held = plexing(makes(), deps)
      const from = at || looking() || deps.opening()
      if (from) void held.view.go(from)
      return held
    },
    called: (held) => plexCalled(words.plex, held.view.neighbourhood.value?.focus?.title ?? ''),
    draws: PlexTab,
    shuts: (held) => {
      held.view.close()
      return true
    },
    offers: words.newPlex,
  }

  /** The note the person is looking at, which is what a question is about. */
  const looking = (): string => front()?.view.here.value ?? ''

  /** What the plex in front could not show, for the window to put up. */
  const trouble = (): string => front()?.view.trouble.value ?? ''

  /** What the plex in front calls a note, and nothing where it names none. */
  const names = (path: string): string => (path ? (front()?.nameOf(path) ?? '') : '')

  /**
   * A note put in front of the person: the plex they are looking at travels
   * there and comes to the front, and a window holding no plex at all opens one
   * on it. The person asked to see the note, so the tab showing it is the tab
   * they are left in.
   */
  const travel = async (path: string) => {
    const one = host.last<Held>(PLEX)
    if (!one) {
      await host.opens(PLEX, path)
      return
    }
    host.shows(one.id)
    await one.held.view.go(path)
  }

  /**
   * Every plex asks for its picture again, following whatever moved: a plex
   * standing on a note that was renamed stands on where it went. One standing
   * nowhere is given the note the vault opens with, which is asked for once for
   * all of them and only while one of them has nowhere to stand.
   */
  const again = async (renamed: readonly Went[] = []) => {
    if (renamed.length) for (const { held } of all()) held.view.follows(renamed)
    if (all().some(({ held }) => !held.view.here.value)) {
      try {
        await deps.first()
      } catch {
        // The next change asks again.
      }
    }
    await Promise.all(
      all().map(({ held }) => {
        const path = held.view.here.value || deps.opening()
        return path ? held.view.go(path) : Promise.resolve()
      }),
    )
  }

  return { kind, looking, trouble, names, travel, again }
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
    carry(id, asking.path, nameOf(asking.path), deps)
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

  return {
    view,
    picture,
    menu,
    creatable: deps.creatable,
    activate,
    made,
    joined,
    opens,
    asks,
    dismiss,
    chose,
    nameOf,
  }
}
