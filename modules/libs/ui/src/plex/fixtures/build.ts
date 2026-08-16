import type { PlexEdge, PlexNeighbourhood, PlexNode, PlexRelatedSeat } from '../model'
import { RELATED_SEATS } from '../model'
import { nameFor } from './names'

export type Counts = Readonly<Partial<Record<PlexRelatedSeat, number>>>

export interface Named {
  readonly id: string
  readonly title: string
}

/** What a typed relationship might be called, so the labels have something to say. */
const RELATION: Record<PlexRelatedSeat, string> = {
  parent: 'is a',
  child: 'contains',
  jump: 'see also',
  sibling: 'contains',
}

/** A neighbourhood of the requested size, for turning knobs against. */
export function build(title: string, counts: Counts): PlexNeighbourhood {
  return around({ id: 'focus', title }, null, counts)
}

/**
 * The same, around a chosen node, with the one it was chosen from seated as a
 * parent. Both keep their identifiers, so the plex recognises them in the new
 * picture and carries them to their seats instead of blinking out and back.
 *
 * Names come from a pool rather than being numbered, because a plex of
 * "Child 1, Child 2" says nothing about whether real titles wrap or collide.
 */
export function around(
  focus: Named,
  from: Named | null,
  counts: Counts,
): PlexNeighbourhood {
  let taken = 0
  const invented: PlexNode[] = RELATED_SEATS.flatMap((seat) =>
    Array.from({ length: counts[seat] ?? 0 }, (_, index) => ({
      id: `${focus.id}/${seat}-${index}`,
      title: nameFor(focus.id, taken++),
      seat,
    })),
  )

  // The node it was chosen from takes the first parent seat, keeping its id.
  //
  // And it is taken out of what was invented, or walking back up would seat it
  // twice: from the parent's side, the node you came down from is one of the
  // children about to be invented, and it already has an identifier. A plex
  // refuses a neighbourhood that seats the same node twice, and rightly.
  const others = from ? invented.filter((node) => node.id !== from.id) : invented

  const related: PlexNode[] = from
    ? [
        { id: from.id, title: from.title, seat: 'parent' },
        ...others.filter((node) => node.seat === 'parent').slice(1),
        ...others.filter((node) => node.seat !== 'parent'),
      ]
    : others

  // A sibling hangs off a parent, not off the focus — it is another of that
  // parent's children. Which parent is a question about relationships, so it
  // is answered here rather than by the plex.
  const firstParent = related.find((node) => node.seat === 'parent')?.id

  const edges: PlexEdge[] = related.flatMap((node) => {
    const title = RELATION[node.seat as PlexRelatedSeat]
    if (node.seat === 'parent' || node.seat === 'jump') {
      return [{ from: node.id, to: focus.id, title }]
    }
    if (node.seat === 'sibling') {
      return firstParent ? [{ from: firstParent, to: node.id, title }] : []
    }
    return [{ from: focus.id, to: node.id, title }]
  })

  return {
    nodes: [{ id: focus.id, title: focus.title, seat: 'focus' }, ...related],
    edges,
  }
}
