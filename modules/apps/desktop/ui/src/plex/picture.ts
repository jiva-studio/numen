/**
 * A neighbourhood of a vault, as something the plex can draw.
 *
 * This is the only file that knows both. The plex is told about nodes with a
 * seat and edges between them; it is not told what a note is, and the schema
 * says nothing about drawing.
 */
import { NeighbourhoodResponseSchema, Seat } from '@numen/protocol'
import type { Neighbourhood } from '../core'
import type {
  EdgeArrow,
  PlexEdge,
  PlexNeighbourhood,
  PlexNode,
  PlexRelatedSeat,
} from '@numen/ui'

/**
 * The schema that builds a neighbourhood. The type is a generated message
 * branded with its own name, so one that did not come off the wire is made
 * with `create(NeighbourhoodSchema, …)`.
 */
export const NeighbourhoodSchema = NeighbourhoodResponseSchema

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
 * parent. Which parent that is comes from the vault, and this file names it.
 *
 * A relationship both notes named is drawn with an arrow at the end away from
 * the focus, the words along the line being the ones the note in focus wrote.
 */
export function asPlex(neighbourhood: Neighbourhood): PlexNeighbourhood {
  const focus: PlexNode = {
    id: neighbourhood.focus?.path ?? '',
    title: neighbourhood.focus?.title ?? '',
    seat: 'focus',
  }

  const nodes: PlexNode[] = [focus]
  const seated: {
    id: string
    seat: PlexRelatedSeat
    label: string
    through: string
    mutual: boolean
  }[] = []
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
      mutual: related.mutual,
    })
  }

  const shown = new Set(nodes.map((node) => node.id))
  const edges: PlexEdge[] = seated.flatMap(({ id, seat, label, through, mutual }) => {
    const line = label ? { label } : {}
    // A mutual line carries an arrow at the end away from the note in focus,
    // whichever end of the line that is. A relationship named at one end only
    // has one wording, and nothing for an arrow to choose between.
    const head = (end: EdgeArrow) => (mutual ? { arrow: end } : {})
    if (seat === 'parent' || seat === 'jump') {
      return [{ from: id, to: focus.id, ...line, ...head('from') }]
    }
    if (seat === 'child') return [{ from: focus.id, to: id, ...line, ...head('to') }]
    // A sibling hangs off the parent it shares, which the answer names. With
    // that parent off the screen it hangs off nothing. Neither end of that line
    // is the note in focus, so no arrow is drawn on it.
    return shown.has(through) ? [{ from: through, to: id, ...line }] : []
  })

  return { nodes, edges }
}
