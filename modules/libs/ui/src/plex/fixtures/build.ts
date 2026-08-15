import type { PlexEdge, PlexNeighbourhood, PlexNode, PlexRelatedRole } from '../model'
import { RELATED_ROLES } from '../model'
import { nameFor } from './names'

export type Counts = Readonly<Partial<Record<PlexRelatedRole, number>>>

export interface Thought {
  readonly id: string
  readonly label: string
}

/** What a typed relationship might be called, so the labels have something to say. */
const RELATION: Record<PlexRelatedRole, string> = {
  parent: 'is a',
  child: 'contains',
  jump: 'see also',
  sibling: 'contains',
}

/** A neighbourhood of the requested size, for turning knobs against. */
export function build(label: string, counts: Counts): PlexNeighbourhood {
  return around({ id: 'focus', label }, null, counts)
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
  focus: Thought,
  from: Thought | null,
  counts: Counts,
): PlexNeighbourhood {
  let taken = 0
  const invented: PlexNode[] = RELATED_ROLES.flatMap((role) =>
    Array.from({ length: counts[role] ?? 0 }, (_, index) => ({
      id: `${focus.id}/${role}-${index}`,
      label: nameFor(focus.id, taken++),
      role,
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
        { id: from.id, label: from.label, role: 'parent' },
        ...others.filter((node) => node.role === 'parent').slice(1),
        ...others.filter((node) => node.role !== 'parent'),
      ]
    : others

  // A sibling hangs off a parent, not off the focus — it is another of that
  // parent's children. Which parent is a question about relationships, so it
  // is answered here rather than by the plex.
  const firstParent = related.find((node) => node.role === 'parent')?.id

  const edges: PlexEdge[] = related.flatMap((node) => {
    const label = RELATION[node.role as PlexRelatedRole]
    if (node.role === 'parent' || node.role === 'jump') {
      return [{ from: node.id, to: focus.id, label }]
    }
    if (node.role === 'sibling') {
      return firstParent ? [{ from: firstParent, to: node.id, label }] : []
    }
    return [{ from: focus.id, to: node.id, label }]
  })

  return {
    nodes: [{ id: focus.id, label: focus.label, role: 'focus' }, ...related],
    edges,
  }
}
