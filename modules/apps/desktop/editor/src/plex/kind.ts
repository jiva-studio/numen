/**
 * What one plex tab holds: where it is standing, and what a gesture in it does.
 * The plex reports the shape of a gesture, and what it comes to is decided here.
 *
 * A gesture names a node by its ticket. This is the edge where a ticket becomes
 * the path the vault is asked about, and past it every note is a path.
 */
import { computed, ref, shallowRef, watch, type Ref } from 'vue'
import type {
  MenuOpening,
  PlexNeighbourhood,
  PlexPart,
  PlexRelatedSeat,
  PlexShowing,
} from '@numen/ui'
import { answerGuard } from '../questions'
import { NEW_NOTE, OFFERED } from './menu'
import { asParts, asPlex, typesIn } from './picture'
import type { View } from './view'
import { ticketing } from './tickets'
import type { Move, NoteHeading, NoteType } from '../core'
import type { Host, Kind } from '../tabs/windowing'
import { PLEX, plexCalled } from '../tabs/workspace'
import PlexTab from './PlexTab.vue'
import { WORDS as words } from './words'

/** Where the menu stands, and the node it was asked for on. */
export interface MenuRequest {
  /** The node it was asked for on, and nothing where it was asked off every node. */
  readonly node: string | null
  readonly at: { x: number; y: number }
  readonly opening: MenuOpening
}

/** Making a note from the picture, and joining two that are already on it. */
export interface PlexEditor {
  make(from: string, seat: PlexRelatedSeat): Promise<unknown>
  join(from: string, to: string, seat: PlexRelatedSeat): Promise<boolean>
}

/** What a plex tab asks of the vault and of the window it is drawn in. */
export interface PlexTabDeps {
  readonly makes: PlexEditor
  /** Whether the window has anything true to draw at all. */
  readonly ready: Readonly<Ref<boolean>>
  /**
   * Whether a node hangs the parts of its note under the box. While it stands
   * false the vault is asked nothing about what a note is divided into, and
   * every node hangs nothing.
   */
  readonly hangs: Readonly<Ref<boolean>>
  /** How many of those parts stand under a node at once. The rest are wound to. */
  readonly parts: Readonly<Ref<number>>
  /**
   * A note opened in a tab of its own, under the name the picture gives it, in
   * the editor made for what it is. A line is a place inside it.
   */
  opens(path: string, title: string, showing: PlexShowing, line?: number): void
  /**
   * What each of the notes asked about is divided into, by the path it was
   * asked about. A note with nothing inside it is absent.
   */
  inside(paths: readonly string[]): Promise<ReadonlyMap<string, readonly NoteHeading[]>>
  /** Something to ask, put in the agent the person was last in. */
  asks(text: string): void
  /**
   * A command asked for on a node, on the note it stands for. One that needs
   * something asks for it in the palette; the rest happen where they stand.
   */
  runs(id: string, path: string, title: string): void
  /** The note the vault opens with, as it was last answered. */
  readonly opening: Readonly<Ref<string>>
  /**
   * The notes the window is dragging over the picture, and none while it
   * drags nothing.
   */
  readonly dragged: Readonly<Ref<readonly string[]>>
  /** What could not be done, in words a person reads. */
  says(text: string): void
  /**
   * A note made at the top of the vault, under a name nothing there carries.
   * The path it landed at, and nothing where none was made.
   */
  writes(): Promise<string>
  /** Asks the vault where it opens, for a plex that has nowhere to stand. */
  first(): Promise<string>
  /** The seats a gesture may make a note in, which the picture draws. */
  readonly creatable: readonly PlexRelatedSeat[]
}

/** What one plex tab holds. */
export type PlexTabState = ReturnType<typeof plexing>

/**
 * The plex tabs of a window, in the order the person was last in them.
 *
 * The plex the person is looking at is the one a note asked for from outside
 * the window is put in front of, and the one a plex opened after it stands
 * beside.
 */
export function plexKind(host: Host, makes: () => View, deps: PlexTabDeps) {
  /** Every plex the window holds, and the one the person was last in. */
  const all = () => host.each<PlexTabState>(PLEX)
  const front = (): PlexTabState | null => host.last<PlexTabState>(PLEX)?.held ?? null

  const kind: Kind<PlexTabState> = {
    kind: PLEX,
    opens: (at) => {
      const held = plexing(makes(), deps)
      const from = at || looking() || deps.opening.value
      if (from) void held.view.go(from)
      return held
    },
    called: (held) => plexCalled(words.plex, held.view.neighbourhood.value?.focus.title ?? ''),
    draws: PlexTab,
    shuts: (held) => {
      held.view.close()
      return true
    },
    at: (held) => {
      const path = held.view.here.value
      return { path, title: (path && held.nameOf(path)) || path }
    },
    attends: (held) => ({ path: held.view.here.value }),
  }

  /** The note the person is looking at, which is what a question is about. */
  const looking = (): string => front()?.view.here.value ?? ''

  /** What the plex in front calls a note, and nothing where it names none. */
  const names = (path: string): string => (path ? (front()?.nameOf(path) ?? '') : '')

  /**
   * A note put in front of the person: the plex they are looking at travels
   * there and comes to the front, and a window holding no plex at all opens one
   * on it.
   */
  const travel = async (path: string) => {
    const one = host.last<PlexTabState>(PLEX)
    if (!one) {
      await host.opens(PLEX, path)
      return
    }
    host.shows(one.id)
    await one.held.view.go(path)
  }

  /**
   * Every plex standing on a note travels to another one. A plex standing
   * anywhere else stays where it is.
   */
  const leaves = async (from: string, to: string) => {
    await Promise.all(
      all()
        .filter(({ held }) => held.view.here.value === from)
        .map(({ held }) => held.view.go(to)),
    )
  }

  /**
   * Every plex asks for its picture again, following whatever moved: a plex
   * standing on a note that was renamed stands on where it went. One standing
   * nowhere is given the note the vault opens with, which is asked for once for
   * all of them and only while one of them has nowhere to stand.
   */
  const again = async (renamed: readonly Move[] = []) => {
    if (renamed.length) for (const { held } of all()) held.follows(renamed)
    if (all().some(({ held }) => !held.view.here.value)) {
      try {
        await deps.first()
      } catch {
        // The next change asks again.
      }
    }
    // The picture is asked for again, and with it what the notes on it are
    // divided into: a note whose headings were edited is drawn in the picture
    // it was already drawn in, and nothing else would ask.
    await Promise.all(
      all().map(async ({ held }) => {
        const path = held.view.here.value || deps.opening.value
        if (path) await held.view.go(path)
        await held.reads()
      }),
    )
  }

  return { kind, looking, names, travel, leaves, again }
}

export function plexing(view: View, deps: PlexTabDeps) {
  /** What this plex calls each note it draws. */
  const tickets = ticketing()

  /**
   * The picture as the plex reads it, and nothing while the window has none.
   * The notes it draws are the notes this plex holds a ticket for.
   */
  const picture = computed<PlexNeighbourhood | null>(() => {
    const around = view.neighbourhood.value
    if (!deps.ready.value || !around) return null
    const drawn = asPlex(around, tickets.of)
    tickets.keeps(drawn.nodes.map((node) => node.id))
    return drawn
  })

  /** Whether the vault has been read and holds no note for this plex to draw. */
  const empty = computed(
    () => deps.ready.value && !view.here.value && !deps.opening.value && !view.neighbourhood.value,
  )

  /**
   * The notes dragged over this picture from elsewhere in the window.
   *
   * A plex with nothing true to draw drags nothing, and the note the plex
   * stands on is left out of what is dragged over it.
   */
  const dragged = computed<readonly string[]>(() => {
    const here = view.here.value
    if (!deps.ready.value || !view.neighbourhood.value || !here) return []
    return deps.dragged.value.filter((path) => path !== here)
  })

  /** The menu on a node, for as long as it stands. */
  const menu = ref<MenuRequest | null>(null)

  /** What each note this plex draws is divided into, by the path it stands at. */
  const parts = shallowRef<ReadonlyMap<string, readonly PlexPart[]>>(new Map())

  /** Which of three each note on the picture is, by the path it stands at. */
  const types = computed<ReadonlyMap<string, NoteType>>(() => {
    const around = view.neighbourhood.value
    return around ? typesIn(around) : new Map()
  })

  /** Which of three the note a ticket names is. */
  const typeOf = (node: string): NoteType => types.value.get(tickets.note(node) ?? '') ?? 'note'

  /** Every note on the picture, the one in focus among them. */
  const drawn = computed<readonly string[]>(() => {
    const around = view.neighbourhood.value
    if (!around) return []
    const paths = [around.focus.path]
    for (const related of around.related) paths.push(related.path)
    return paths.filter(Boolean)
  })

  /** Two answers can be in flight — a change followed while a travel is still out. */
  const reading = answerGuard()

  /**
   * What the notes on the picture are divided into, asked for all of them at
   * once and kept until the next question. A vault that cannot answer leaves
   * every node hanging nothing.
   */
  const reads = async () => {
    // A deck divides into its cards and a stencil into its faces, and neither
    // is a part of prose. Only an ordinary note is asked about.
    const paths = drawn.value.filter((path) => (types.value.get(path) ?? 'note') === 'note')
    const mine = reading.ask()
    if (!deps.hangs.value || paths.length === 0) {
      parts.value = new Map()
      return
    }
    try {
      const found = await deps.inside(paths)
      if (!mine.current) return
      parts.value = new Map([...found].map(([path, held]) => [path, asParts(held)]))
    } catch {
      // What hangs under a node is the picture saying more about notes already
      // drawn. A vault that cannot answer leaves them hanging nothing, and the
      // next picture asks again.
      if (mine.current) parts.value = new Map()
    }
  }

  // The picture the plex travels to is a new set of notes. A note whose
  // headings were edited is drawn in the same picture, and `again` asks.
  watch(drawn, () => void reads(), { immediate: true })

  // The setting turned: what each node hangs is asked for again, so a picture
  // already drawn hangs what the setting now says.
  watch(deps.hangs, () => void reads())

  /** The parts of the note a ticket names, and none while the setting is off. */
  const partsOf = (node: string): readonly PlexPart[] =>
    deps.hangs.value ? (parts.value.get(tickets.note(node) ?? '') ?? []) : []

  /** A node chosen: the plex travels there, and the picture is asked for again. */
  const activate = (node: string) => {
    const path = tickets.note(node)
    if (path) void view.go(path)
  }

  /**
   * A note made in a seat of another one. It is in the index by the time the
   * answer arrives, so the picture is asked for again and it is drawn in it.
   *
   * The plex stands on the note it was made from: a neighbourhood is one seat
   * deep, and that is the seat the new note sits in.
   */
  const made = async (from: string, seat: PlexRelatedSeat) => {
    const path = tickets.note(from)
    if (path && (await deps.makes.make(path, seat))) await view.go(path)
  }

  /**
   * Two notes the person drew a line between. The plex stands on the note the
   * line was drawn from, which is the one the link is written in.
   */
  const joined = async (from: string, to: string, seat: PlexRelatedSeat) => {
    const one = tickets.note(from)
    const other = tickets.note(to)
    if (one && other && (await deps.makes.join(one, other, seat))) await view.go(one)
  }

  /**
   * Notes dragged in from the vault and let go over the picture. They are
   * paths already, having never been drawn as nodes of this picture.
   *
   * Each takes the one seat in the note the plex stands on, which is the one
   * every link is written in. A note the vault refused stays unjoined and is
   * said; the picture is asked for again once any of them was written.
   */
  const brought = async (dragged: readonly string[], seat: PlexRelatedSeat) => {
    const here = view.here.value
    if (!here) return

    const refused: string[] = []
    let written = false

    for (const path of dragged) {
      if (path === here) continue
      if (await deps.makes.join(here, path, seat)) written = true
      else refused.push(nameOf(path))
    }

    if (refused.length > 0) deps.says(`${words.refused} ${refused.join(', ')}`)
    if (written) await view.go(here)
  }

  /** A note opened where the person asked for it, called what the picture calls it. */
  const opens = (node: string, showing: PlexShowing = 'here') => {
    const path = tickets.note(node)
    if (path) deps.opens(path, nameOf(path), showing)
  }

  /**
   * A part of a node chosen: the note it stands in is put in front of the
   * person, and the keyboard goes to the line the part stands on.
   *
   * The plex stays where it is standing. A part is somewhere in a note, and
   * going to one is not travelling to the note as a thing.
   */
  const entered = (node: string, part: string) => {
    const path = tickets.note(node)
    // A part is named by the line it stands on, and anything else names the
    // ones that did not fit.
    if (!path || !/^\d+$/.test(part)) return
    deps.opens(path, nameOf(path), 'here', Number(part))
  }

  /**
   * A note made at the top of the vault, which the plex then stands on. It is
   * in the index by the time the answer arrives, so the picture drawn around it
   * is the one the vault holds.
   */
  const writes = async () => {
    const path = await deps.writes()
    if (path) await view.go(path)
  }

  /** A menu asked for on a node or off every node, and one put away. */
  const asks = (asked: MenuRequest) => {
    menu.value = asked
  }
  const dismiss = () => {
    menu.value = null
  }

  /** An item chosen in the menu, on the note it was asked for on. */
  const chose = (id: string) => {
    const on = menu.value
    menu.value = null
    if (!on) return
    if (on.node === null) {
      if (id === NEW_NOTE) void writes()
      return
    }
    if (!OFFERED.has(id)) return
    const path = tickets.note(on.node)
    if (path) deps.runs(id, path, nameOf(path))
  }

  /**
   * Notes that moved. The plex follows the one it stands on, and every note it
   * draws keeps the ticket it holds.
   */
  const follows = (renamed: readonly Move[]) => {
    view.follows(renamed)
    tickets.moved(renamed)
  }

  /**
   * What a note is called, as the picture has it. A note the picture does not
   * name is called by the file it is filed under.
   */
  const nameOf = (path: string): string => {
    const around = view.neighbourhood.value
    if (around?.focus.path === path && around.focus.title) return around.focus.title
    const near = around?.related.find((one) => one.path === path)
    return near?.title || (path.split('/').pop() ?? path).replace(/\.md$/, '')
  }

  return {
    view,
    picture,
    empty,
    dragged,
    menu,
    creatable: deps.creatable,
    typeOf,
    partsOf,
    mostParts: deps.parts,
    reads,
    entered,
    activate,
    made,
    joined,
    brought,
    opens,
    writes,
    asks,
    dismiss,
    chose,
    follows,
    nameOf,
  }
}
