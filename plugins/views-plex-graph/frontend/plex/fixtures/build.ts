import type { PlexEdge } from '../lib/edge'
import type { PlexNeighbourhood } from '../lib/neighbourhood'
import type { PlexNode } from '../lib/node'
import { RELATION_NAMES, type PlexRelation } from '../lib/relation'
import { createName } from './names'

export type RelationCounts = Readonly<Partial<Record<PlexRelation, number>>>

export interface TitledNode {
  readonly id: string
  readonly title: string
}

/** What a typed relationship might be called, so the labels have something to say. */
const RELATION: Record<PlexRelation, string> = {
  parent: 'is a',
  child: 'contains',
  jump: 'see also',
  sibling: 'contains',
}

/** A neighbourhood of the requested size, for turning knobs against. */
export function build(title: string, counts: RelationCounts): PlexNeighbourhood {
  return buildAround({ id: 'focus', title }, null, counts)
}

/**
 * The same, around a chosen node, with the node it was chosen from placed as a
 * parent. Both keep their identifiers, so the plex recognizes them in the new
 * layout and moves them to their relations instead of removing and re-adding them.
 *
 * Names come from a pool and are not numbered, because a plex of
 * "Child 1, Child 2" does not show whether real titles wrap or collide.
 */
export function buildAround(
  focus: TitledNode,
  from: TitledNode | null,
  counts: RelationCounts,
): PlexNeighbourhood {
  let nameIndex = 0
  const invented: PlexNode[] = RELATION_NAMES.flatMap((relation) =>
    Array.from({ length: counts[relation] ?? 0 }, (_, index) => ({
      id: `${focus.id}/${relation}-${index}`,
      title: createName(focus.id, nameIndex++),
      relation,
    })),
  )

  // The node it was chosen from takes the first parent relation, keeping its id.
  //
  // It is also removed from the generated nodes: from the parent's side, the
  // node navigated from is one of the children about to be generated, and it
  // already has an identifier. A plex rejects a neighbourhood that contains the
  // same node twice.
  const others = from ? invented.filter((node) => node.id !== from.id) : invented

  const related: PlexNode[] = from
    ? [
        { id: from.id, title: from.title, relation: 'parent' },
        ...others.filter((node) => node.relation === 'parent').slice(1),
        ...others.filter((node) => node.relation !== 'parent'),
      ]
    : others

  // A sibling hangs off a parent, not off the focus — it is another of that
  // parent's children. Which parent is a question about relationships, so it
  // is answered here and not by the plex.
  const firstParent = related.find((node) => node.relation === 'parent')?.id

  const edges: PlexEdge[] = related.flatMap((node): PlexEdge[] => {
    const label = RELATION[node.relation as PlexRelation]
    if (node.relation === 'parent' || node.relation === 'jump') {
      return [{ from: node.id, to: focus.id, label }]
    }
    if (node.relation === 'sibling') {
      return firstParent ? [{ from: firstParent, to: node.id, label }] : []
    }
    return [{ from: focus.id, to: node.id, label }]
  })

  return {
    nodes: [{ id: focus.id, title: focus.title, relation: 'focus' }, ...related],
    edges,
  }
}
