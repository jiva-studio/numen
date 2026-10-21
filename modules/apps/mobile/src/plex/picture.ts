/**
 * A neighbourhood of a vault, as something the plex can draw.
 *
 * This is the only file that knows both. The plex is told about nodes with a
 * relation and edges between them; it is not told what a note is, and the schema
 * says nothing about drawing.
 */
import { Relation } from '@numen/protocol'
import type { GetNeighbourhoodResponse } from '@numen/protocol'
import type { EdgeArrow, PlexEdge, PlexNeighbourhood, PlexNode, PlexRelation } from '@numen/ui'

const RELATION_BY_WIRE: Record<Relation, PlexRelation | null> = {
  [Relation.UNSPECIFIED]: null,
  [Relation.PARENT]: 'parent',
  [Relation.CHILD]: 'child',
  [Relation.JUMP]: 'jump',
  [Relation.SIBLING]: 'sibling',
}

/**
 * A node is drawn under the path its note holds, and so is each end of an edge.
 *
 * A sibling's edge does not touch the focus at all: it hangs off the parent the
 * two share.
 */
export function convertNeighbourhood(neighbourhood: GetNeighbourhoodResponse): PlexNeighbourhood {
  const focus = getFocusNode(neighbourhood.focus)

  const nodes: PlexNode[] = [focus]
  const visibleIds = new Set<string>([focus.id])
  const edges: PlexEdge[] = []
  // Stryker disable next-line ArrayDeclaration: the mutated value is a string element; destructuring it gives an undefined through, which is in no shown set, so the loop adds nothing
  const hanging: { id: string; label: string; through: string }[] = []

  for (const related of neighbourhood.related) {
    const relation = RELATION_BY_WIRE[related.relation]
    if (!relation || !related.note) continue
    const id = related.note.path
    visibleIds.add(id)
    nodes.push({ id, title: related.note.title, relation })

    if (relation === 'sibling') {
      hanging.push({ id, label: related.label, through: related.through })
      continue
    }
    edges.push(getEdge(id, focus.id, relation, related))
  }

  for (const { id, label, through } of hanging) {
    if (!visibleIds.has(through)) continue
    edges.push({ from: through, to: id, ...(label ? { label } : {}) })
  }

  return { nodes, edges }
}

type RelatedNote = GetNeighbourhoodResponse['related'][number]

/** The note the neighbourhood is drawn around, under an empty path when there is none. */
const getFocusNode = (focus: GetNeighbourhoodResponse['focus']): PlexNode => ({
  id: focus?.path ?? '',
  title: focus?.title ?? '',
  relation: 'focus',
})

/**
 * A parent and a jump run into the focus, and a child runs out of it. A
 * relationship both notes named carries an arrow at the end away from the focus.
 */
const getEdge = (
  id: string,
  focus: string,
  relation: PlexRelation,
  note: RelatedNote,
): PlexEdge => {
  const line = note.label ? { label: note.label } : {}
  const getArrow = (end: EdgeArrow) => (note.isMutual ? { arrow: end } : {})
  if (relation === 'child') return { from: focus, to: id, ...line, ...getArrow('to') }
  return { from: id, to: focus, ...line, ...getArrow('from') }
}
