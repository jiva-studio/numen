/**
 * A neighbourhood of a vault, as something the plex can draw.
 *
 * This is the only file that knows both. The plex is told about nodes with a
 * seat and edges between them; it is not told what a note is, and the schema
 * says nothing about drawing.
 */
import { Seat } from '@numen/protocol'
import type { GetNeighbourhoodResponse } from '@numen/protocol'
import type { EdgeArrow, PlexEdge, PlexNeighbourhood, PlexNode, PlexRelatedSeat } from '@numen/ui'

const seats: Record<Seat, PlexRelatedSeat | null> = {
  [Seat.UNSPECIFIED]: null,
  [Seat.PARENT]: 'parent',
  [Seat.CHILD]: 'child',
  [Seat.JUMP]: 'jump',
  [Seat.SIBLING]: 'sibling',
}

/**
 * A node is drawn under the path its note holds, and so is each end of an edge.
 *
 * A sibling's edge does not touch the focus at all: it hangs off the parent the
 * two share.
 */
export function asPlex(neighbourhood: GetNeighbourhoodResponse): PlexNeighbourhood {
  const focus = getFocusNode(neighbourhood.focus)

  const nodes: PlexNode[] = [focus]
  const shown = new Set<string>([focus.id])
  const edges: PlexEdge[] = []
  const hanging: { id: string; label: string; through: string }[] = []

  for (const related of neighbourhood.related) {
    const seat = seats[related.seat]
    if (!seat || !related.note) continue
    const id = related.note.path
    shown.add(id)
    nodes.push({ id, title: related.note.title, seat })

    if (seat === 'sibling') {
      hanging.push({ id, label: related.label, through: related.through })
      continue
    }
    edges.push(getEdge(id, focus.id, seat, related))
  }

  for (const { id, label, through } of hanging) {
    if (!shown.has(through)) continue
    edges.push({ from: through, to: id, ...(label ? { label } : {}) })
  }

  return { nodes, edges }
}

type RelatedNote = GetNeighbourhoodResponse['related'][number]

/** The note the neighbourhood is drawn around, under an empty path when there is none. */
const getFocusNode = (focus: GetNeighbourhoodResponse['focus']): PlexNode => ({
  id: focus?.path ?? '',
  title: focus?.title ?? '',
  seat: 'focus',
})

/**
 * A parent and a jump run into the focus, and a child runs out of it. A
 * relationship both notes named carries an arrow at the end away from the focus.
 */
const getEdge = (
  id: string,
  focus: string,
  seat: PlexRelatedSeat,
  note: RelatedNote,
): PlexEdge => {
  const line = note.label ? { label: note.label } : {}
  const getArrow = (end: EdgeArrow) => (note.mutual ? { arrow: end } : {})
  if (seat === 'child') return { from: focus, to: id, ...line, ...getArrow('to') }
  return { from: id, to: focus, ...line, ...getArrow('from') }
}
