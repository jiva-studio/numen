/**
 * Tab state and interactions for an open plex graph tab.
 */
import { computed, ref } from 'vue'
import type {
  PlexNeighbourhood,
  PlexPart,
  PlexRelatedSeat,
  PlexShowing,
} from '@numen/ui'
import type { Move, NoteType } from '../shared/core'
import { fileOf } from '../shared/paths'
import { NEW_NOTE, OFFERED } from './menu'
import { asPlex, typesIn } from './picture'
import { createTickets } from './tickets'
import type { PlexView } from './view'
import { WORDS as words } from './words'
import type { MenuRequest, PlexTabDeps, PlexTabState } from './types'
import { usePlexParts } from './parts'

export function usePlexTab(view: PlexView, deps: PlexTabDeps): PlexTabState {
  const tickets = createTickets()

  const picture = computed<PlexNeighbourhood | null>(() => {
    const around = view.neighbourhood.value
    if (!deps.ready.value || !around) return null
    const drawn = asPlex(around, tickets.of)
    tickets.keeps(drawn.nodes.map((node) => node.id))
    return drawn
  })

  const empty = computed(
    () => deps.ready.value && !view.here.value && !deps.opening.value && !view.neighbourhood.value,
  )

  const dragged = computed<readonly string[]>(() => {
    const here = view.here.value
    if (!deps.ready.value || !view.neighbourhood.value || !here) return []
    return deps.dragged.value.filter((path) => path !== here)
  })

  const menu = ref<MenuRequest | null>(null)

  const types = computed<ReadonlyMap<string, NoteType>>(() => {
    const around = view.neighbourhood.value
    return around ? typesIn(around) : new Map()
  })

  const typeOf = (node: string): NoteType => types.value.get(tickets.note(node) ?? '') ?? 'note'

  const drawn = computed<readonly string[]>(() => {
    const around = view.neighbourhood.value
    if (!around) return []
    const paths = [around.focus.path]
    for (const related of around.related) paths.push(related.path)
    return paths.filter(Boolean)
  })

  const { readParts, getParts } = usePlexParts(drawn, types, deps, tickets)

  const activate = (node: string) => {
    const path = tickets.note(node)
    if (path) void view.go(path)
  }

  const made = async (from: string, seat: PlexRelatedSeat) => {
    const path = tickets.note(from)
    if (path && (await deps.makes.make(path, seat))) await view.go(path)
  }

  const joined = async (from: string, to: string, seat: PlexRelatedSeat) => {
    const one = tickets.note(from)
    const other = tickets.note(to)
    if (one && other && (await deps.makes.join(one, other, seat))) await view.go(one)
  }

  const brought = async (draggedNodes: readonly string[], seat: PlexRelatedSeat) => {
    const here = view.here.value
    if (!here) return

    const refused: string[] = []
    let written = false

    for (const path of draggedNodes) {
      if (path === here) continue
      if (await deps.makes.join(here, path, seat)) written = true
      else refused.push(nameOf(path))
    }

    if (refused.length > 0) deps.says(`${words.refused} ${refused.join(', ')}`)
    if (written) await view.go(here)
  }

  const opens = (node: string, showing: PlexShowing = 'here') => {
    const path = tickets.note(node)
    if (path) deps.opens(path, nameOf(path), showing)
  }

  const entered = (node: string, part: string) => {
    const path = tickets.note(node)
    if (!path || !/^\d+$/.test(part)) return
    deps.opens(path, nameOf(path), 'here', Number(part))
  }

  const writes = async () => {
    const path = await deps.writes()
    if (path) await view.go(path)
  }

  const asks = (asked: MenuRequest) => {
    menu.value = asked
  }

  const dismiss = () => {
    menu.value = null
  }

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

  const follows = (renamed: readonly Move[]) => {
    view.follows(renamed)
    tickets.moved(renamed)
  }

  const nameOf = (path: string): string => {
    const around = view.neighbourhood.value
    if (around?.focus.path === path && around.focus.title) return around.focus.title
    const near = around?.related.find((one) => one.path === path)
    return near?.title || fileOf(path).replace(/\.md$/, '')
  }

  return {
    view,
    picture,
    empty,
    dragged,
    menu,
    creatable: deps.creatable,
    typeOf,
    partsOf: getParts,
    mostParts: deps.parts,
    reads: readParts,
    readParts,
    entered,
    openPart: entered,
    activate,
    made,
    createNode: made,
    joined,
    joinNodes: joined,
    brought,
    bringNodes: brought,
    opens,
    openNode: opens,
    writes,
    createNote: writes,
    asks,
    openMenu: asks,
    dismiss,
    chose,
    chooseMenuItem: chose,
    follows,
    followMoves: follows,
    nameOf,
    getName: nameOf,
  }
}
