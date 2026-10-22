/**
 * A neighbourhood of a vault, as something the plex can draw.
 *
 * This is the only file that knows both. The plex is told about nodes with a
 * relation and edges between them; it is not told what a note is, and the schema
 * says nothing about drawing.
 */
import type { Neighbourhood, NoteHeading, NoteRelation, NoteType } from '../types'
import type { EdgeArrow, PlexEdge, PlexNeighbourhood, PlexNode, PlexHeading } from '@numen/ui'

/**
 * plex draws a deck and a stencil as what they are.
 */
export function getNoteTypes(neighbourhood: Neighbourhood): ReadonlyMap<string, NoteType> {
  const found = new Map<string, NoteType>()
  const focus = neighbourhood.focus.path
  if (focus) found.set(focus, neighbourhood.focusType)
  for (const related of neighbourhood.related) found.set(related.path, related.type)
  return found
}

/** A relationship both notes named is drawn with an arrow at the end away from the focus. */
function getEdgeArrow(relation: NoteRelation, isMutual: boolean): EdgeArrow | undefined {
  if (isMutual) return relation === 'child' ? 'to' : 'from'
  return undefined
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
 */
export function convertNeighbourhood(
  neighbourhood: Neighbourhood,
  ticket: (path: string) => string,
): PlexNeighbourhood {
  const focus: PlexNode = {
    id: ticket(neighbourhood.focus.path),
    title: neighbourhood.focus.title,
    relation: 'focus',
  }

  const nodes: PlexNode[] = [focus]
  // Stryker disable ArrayDeclaration: an entry that is not a note with a relation has no relation, and no edge is drawn for it.
  const items: {
    id: string
    relation: NoteRelation
    label: string
    through: string
    isMutual: boolean
  }[] = []
  // Stryker restore ArrayDeclaration
  /** The path of every note this picture draws, the note in focus included. */
  const visiblePaths = new Set<string>([neighbourhood.focus.path])
  for (const related of neighbourhood.related) {
    const id = ticket(related.path)
    visiblePaths.add(related.path)
    nodes.push({ id, title: related.title, relation: related.relation })
    items.push({
      id,
      relation: related.relation,
      // What the person wrote on the link. A line with nothing written on it
      // carries nothing: a word put there by the application would be read as
      // one they had written themselves.
      label: related.label,
      through: related.through,
      isMutual: related.isMutual,
    })
  }

  const edges: PlexEdge[] = items.flatMap(({ id, relation, label, through, isMutual }) => {
    const line = label ? { label } : {}
    const arrow = getEdgeArrow(relation, isMutual)
    const head = arrow ? { arrow } : {}

    if (relation === 'parent' || relation === 'jump') {
      return [{ from: id, to: focus.id, ...line, ...head }]
    }
    if (relation === 'child') {
      return [{ from: focus.id, to: id, ...line, ...head }]
    }
    return visiblePaths.has(through) ? [{ from: ticket(through), to: id, ...line }] : []
  })

  return { nodes, edges }
}

/**
 * The headings of a note as the lines its node hangs.
 *
 * A heading is named by the line it relies on: that is what taking someone to it
 * needs, and it tells two headings of one wording apart.
 */
export function convertHeadings(headings: readonly NoteHeading[]): PlexHeading[] {
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
export function areNeighbourhoodsEqual(
  one: Neighbourhood | null,
  other: Neighbourhood | null,
): boolean {
  if (one === null || other === null) return one === other
  if (one.focus.path !== other.focus.path) return false
  if (one.focus.title !== other.focus.title) return false
  if (one.focusType !== other.focusType) return false
  if (one.related.length !== other.related.length) return false

  return one.related.every((related, at) => {
    const comparison = other.related[at]!
    return (
      related.path === comparison.path &&
      related.title === comparison.title &&
      related.type === comparison.type &&
      related.relation === comparison.relation &&
      related.label === comparison.label &&
      related.through === comparison.through &&
      related.isMutual === comparison.isMutual
    )
  })
}
