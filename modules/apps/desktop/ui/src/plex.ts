/**
 * A neighbourhood of a vault, as something the plex can draw.
 *
 * This is the only file that knows both. The plex is told about nodes with a
 * seat and edges between them; it is not told what a note is, and the schema
 * says nothing about drawing.
 */
import { Seat, type NeighbourhoodResponse } from '@numen/protocol'
import type { PlexEdge, PlexNeighbourhood, PlexNode, PlexRelatedSeat } from '@numen/ui'

export type Neighbourhood = NeighbourhoodResponse

const seats: Record<Seat, PlexRelatedSeat | null> = {
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
    title: neighbourhood.focus?.title ?? '',
    seat: 'focus',
  }

  const nodes: PlexNode[] = [focus]
  const seated: { id: string; seat: PlexRelatedSeat; label: string; through: string }[] = []
  for (const related of neighbourhood.related) {
    const seat = seats[related.seat]
    if (!seat || !related.note) continue
    nodes.push({ id: related.note.path, title: related.note.title, seat })
    seated.push({
      id: related.note.path,
      seat,
      // What the person wrote on the link. A line with nothing written on it
      // carries nothing: a word put there by the application would be read as
      // one they had written themselves.
      label: related.label,
      through: related.through,
    })
  }

  const shown = new Set(nodes.map((node) => node.id))
  const edges: PlexEdge[] = seated.flatMap(({ id, seat, label, through }) => {
    const line = label ? { label } : {}
    if (seat === 'parent' || seat === 'jump') return [{ from: id, to: focus.id, ...line }]
    if (seat === 'child') return [{ from: focus.id, to: id, ...line }]
    // A sibling hangs off the parent it shares, which the answer names. With
    // that parent off the screen it hangs off nothing.
    return shown.has(through) ? [{ from: through, to: id, ...line }] : []
  })

  return { nodes, edges }
}
