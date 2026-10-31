/**
 * A neighbourhood of a vault, as something the plex can draw.
 *
 * This is the only file that depends on both sides. The plex receives nodes with
 * a relation and edges between them and has no concept of a note; the schema
 * has no concept of drawing.
 */
import type { LinkMarker } from '@numen/plugin-sdk'
import type { Neighbourhood, NoteHeading, NoteRelation, NoteType } from '../types'
import type { EdgeArrow, PlexEdge, PlexNeighbourhood, PlexNode, PlexHeading } from '../plex'

/**
 * plex draws a note of a plugin-declared type with that type's icon.
 */
export function getNoteTypes(neighbourhood: Neighbourhood): ReadonlyMap<string, NoteType> {
  const found = new Map<string, NoteType>()
  const focus = neighbourhood.focus.path
  if (focus) found.set(focus, neighbourhood.focusType)
  for (const related of neighbourhood.related) found.set(related.path, related.type)
  return found
}

/**
 * The ends of an edge that carry an arrowhead. The marker of the record the focus
 * writes is drawn at the neighbour, and the marker of the record the neighbour
 * writes at the focus. A parent or a jump edge runs from the neighbour to the focus,
 * and a child edge from the focus to the neighbour. A sibling hangs off a shared
 * parent, and no record of the focus is drawn on it.
 */
function getEdgeArrows(
  relation: NoteRelation,
  markers: { readonly marker: LinkMarker | null; readonly reverseMarker: LinkMarker | null },
): readonly EdgeArrow[] {
  if (relation === 'sibling') return []
  const atNeighbour: EdgeArrow = relation === 'child' ? 'to' : 'from'
  const atFocus: EdgeArrow = relation === 'child' ? 'from' : 'to'
  return [
    ...(markers.marker === 'arrow' ? [atNeighbour] : []),
    ...(markers.reverseMarker === 'arrow' ? [atFocus] : []),
  ]
}

/**
 * A node is drawn under the token its note holds, and so is each end of an
 * edge. The caller supplies the token of a note.
 *
 * An edge runs the way the relationship runs, and a sibling's does not touch
 * the focus at all: it is another of a parent's children, so it hangs off that
 * parent. Which parent that is comes from the vault, and this file names it.
 *
 * An arrowhead is drawn at each end whose record carries a marker, and nowhere else.
 */
export function convertNeighbourhood(
  neighbourhood: Neighbourhood,
  token: (path: string) => string,
): PlexNeighbourhood {
  const focus: PlexNode = {
    id: token(neighbourhood.focus.path),
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
    marker: LinkMarker | null
    reverseMarker: LinkMarker | null
  }[] = []
  // Stryker restore ArrayDeclaration
  /** The path of every note this picture draws, the note in focus included. */
  const visiblePaths = new Set<string>([neighbourhood.focus.path])
  for (const related of neighbourhood.related) {
    const id = token(related.path)
    visiblePaths.add(related.path)
    nodes.push({ id, title: related.title, relation: related.relation })
    items.push({
      id,
      relation: related.relation,
      // The text on the link. A line with no text carries nothing: a word added
      // by the application would be read as written by the user.
      label: related.label,
      through: related.through,
      marker: related.marker,
      reverseMarker: related.reverseMarker,
    })
  }

  const edges: PlexEdge[] = items.flatMap(
    ({ id, relation, label, through, marker, reverseMarker }) => {
      const line = label ? { label } : {}
      const arrows = getEdgeArrows(relation, { marker, reverseMarker })
      const head = arrows.length > 0 ? { arrows } : {}

      if (relation === 'parent' || relation === 'jump') {
        return [{ from: id, to: focus.id, ...line, ...head }]
      }
      if (relation === 'child') {
        return [{ from: focus.id, to: id, ...line, ...head }]
      }
      return visiblePaths.has(through) ? [{ from: token(through), to: id, ...line }] : []
    },
  )

  return { nodes, edges }
}

/**
 * The headings of a note as the lines its node hangs.
 *
 * A heading is identified by the line it depends on: that is what navigating
 * to it needs, and it distinguishes two headings of the same wording.
 */
export function convertHeadings(headings: readonly NoteHeading[]): PlexHeading[] {
  return headings.map((heading) => ({
    id: `${heading.line}`,
    text: heading.text,
    level: heading.level,
  }))
}

/**
 * Whether two neighbourhoods draw one graph: the note in focus, and every
 * note joined to it in arrival order, each with the label of the line between
 * them.
 *
 * A vault that changed elsewhere returns a neighbourhood equal to the one on
 * screen, and a graph equal to the one on screen is left unchanged.
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

  return one.related.every((related, index) => {
    const comparison = other.related[index]!
    return (
      related.path === comparison.path &&
      related.title === comparison.title &&
      related.type === comparison.type &&
      related.relation === comparison.relation &&
      related.label === comparison.label &&
      related.through === comparison.through &&
      related.marker === comparison.marker &&
      related.reverseMarker === comparison.reverseMarker
    )
  })
}
