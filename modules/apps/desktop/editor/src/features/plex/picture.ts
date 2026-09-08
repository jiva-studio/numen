/**
 * A neighbourhood of a vault, as something the plex can draw.
 *
 * This is the only file that knows both. The plex is told about nodes with a
 * seat and edges between them; it is not told what a note is, and the schema
 * says nothing about drawing.
 */
import type { Neighbourhood, NoteHeading, NoteType, Seat } from '../../shared/core'
import type { EdgeArrow, PlexEdge, PlexNeighbourhood, PlexNode, PlexPart } from '@numen/ui'

/**
 * Which of three each note on a neighbourhood is, by the path it stands at. A
 * plex draws a deck and a stencil as what they are.
 */
export function typesIn(neighbourhood: Neighbourhood): ReadonlyMap<string, NoteType> {
  const found = new Map<string, NoteType>()
  const focus = neighbourhood.focus.path
  if (focus) found.set(focus, neighbourhood.focusType)
  for (const related of neighbourhood.related) found.set(related.path, related.type)
  return found
}

/**
 * A node is drawn under the ticket its note holds, and so is each end of an
 * edge. The caller says what a note's ticket is.
 *
 * An edge runs the way the relationship runs, and a sibling's does not touch
 * the focus at all: it is another of a parent's children, so it hangs off that
 * parent. Which parent that is comes from the vault, and this file names it.
 *
 * A relationship both notes named is drawn with an arrow at the end away from
 * the focus, the words along the line being the ones the note in focus wrote.
 */
export function asPlex(
  neighbourhood: Neighbourhood,
  ticket: (path: string) => string,
): PlexNeighbourhood {
  const focus: PlexNode = {
    id: ticket(neighbourhood.focus.path),
    title: neighbourhood.focus.title,
    seat: 'focus',
  }

  const nodes: PlexNode[] = [focus]
  const seated: {
    id: string
    seat: Seat
    label: string
    through: string
    mutual: boolean
  }[] = []
  /** The path of every note this picture draws, the note in focus included. */
  const shown = new Set<string>([neighbourhood.focus.path])
  for (const related of neighbourhood.related) {
    const id = ticket(related.path)
    shown.add(related.path)
    nodes.push({ id, title: related.title, seat: related.seat })
    seated.push({
      id,
      seat: related.seat,
      // What the person wrote on the link. A line with nothing written on it
      // carries nothing: a word put there by the application would be read as
      // one they had written themselves.
      label: related.label,
      through: related.through,
      mutual: related.mutual,
    })
  }

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
    return shown.has(through) ? [{ from: ticket(through), to: id, ...line }] : []
  })

  return { nodes, edges }
}

/**
 * The headings of a note as the parts its node hangs.
 *
 * A part is named by the line it stands on: that is what taking someone to it
 * needs, and it tells two headings of one wording apart.
 */
export function asParts(headings: readonly NoteHeading[]): PlexPart[] {
  return headings.map((heading) => ({
    id: `${heading.line}`,
    text: heading.text,
    level: heading.level,
  }))
}

/**
 * Whether two neighbourhoods draw one picture: the note in focus, and every
 * note joined to it in the order they arrived, each with what the line between
 * them says.
 *
 * A vault that changed somewhere else answers with a neighbourhood equal to the
 * one on screen, and a picture equal to the one on screen is left standing.
 */
export function alike(one: Neighbourhood | null, other: Neighbourhood | null): boolean {
  if (one === null || other === null) return one === other
  if (one.focus.path !== other.focus.path) return false
  if (one.focus.title !== other.focus.title) return false
  if (one.focusType !== other.focusType) return false
  if (one.related.length !== other.related.length) return false

  return one.related.every((related, at) => {
    const against = other.related[at]
    return (
      against !== undefined &&
      related.path === against.path &&
      related.title === against.title &&
      related.type === against.type &&
      related.seat === against.seat &&
      related.label === against.label &&
      related.through === against.through &&
      related.mutual === against.mutual
    )
  })
}
