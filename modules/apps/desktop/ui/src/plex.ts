/**
 * A neighbourhood of a vault, as something the plex can draw.
 *
 * This is the only file that knows both. The plex is told about nodes with a
 * seat and edges between them; it is not told what a note is, and the schema
 * says nothing about drawing.
 */
import { Seat, type NeighbourhoodResponse } from '@numen/protocol'
import { seatOf, type PlexEdge, type PlexNeighbourhood, type PlexNode, type PlexRelatedRole } from '@numen/ui'

export type Neighbourhood = NeighbourhoodResponse

const seats: Record<Seat, PlexRelatedRole | null> = {
  [Seat.UNSPECIFIED]: null,
  [Seat.PARENT]: 'parent',
  [Seat.CHILD]: 'child',
  [Seat.JUMP]: 'jump',
  [Seat.SIBLING]: 'sibling',
}

/**
 * A note is addressed by the path it is filed under: every note has one, and a
 * note written outside the application carries no identifier.
 *
 * An edge runs the way the relationship runs, and a sibling's does not touch
 * the focus at all: it is another of a parent's children, so it hangs off that
 * parent. Which parent is a question about the vault, which is why it is
 * answered here and not by the plex.
 */
export function asPlex(neighbourhood: Neighbourhood): PlexNeighbourhood {
  const focus: PlexNode = {
    id: neighbourhood.focus?.path ?? '',
    label: neighbourhood.focus?.title ?? '',
    role: 'focus',
  }

  const nodes: PlexNode[] = [focus]
  const seated: { id: string; role: PlexRelatedRole; label: string; through: string }[] = []
  for (const related of neighbourhood.related) {
    const role = seats[related.seat]
    if (!role || !related.note) continue
    nodes.push({ id: related.note.path, label: related.note.title, role })
    seated.push({
      id: related.note.path,
      role,
      // What the person wrote on the link, and the name of the seat when they
      // wrote nothing.
      label: related.label || seatOf(role),
      through: related.through,
    })
  }

  const shown = new Set(nodes.map((node) => node.id))
  const edges: PlexEdge[] = seated.flatMap(({ id, role, label, through }) => {
    if (role === 'parent' || role === 'jump') return [{ from: id, to: focus.id, label }]
    if (role === 'child') return [{ from: focus.id, to: id, label }]
    // A sibling hangs off the parent it shares, which the answer names. With
    // that parent off the screen it hangs off nothing.
    return shown.has(through) ? [{ from: through, to: id, label }] : []
  })

  return { nodes, edges }
}
