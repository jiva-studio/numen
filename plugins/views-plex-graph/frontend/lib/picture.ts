/**
 * A neighbourhood of a vault, as something the plex can draw.
 *
 * This is the only file that depends on both sides. The plex receives nodes with
 * a relation and edges between them and has no concept of a note; the schema
 * has no concept of drawing.
 */
import type { Connection, ConnectionSide } from '@numen/plugin-sdk'
import type { Neighbourhood, NoteHeading, NoteType } from '../types'
import type {
  EdgeArrow,
  EdgeLabel,
  PlexEdge,
  PlexNeighbourhood,
  PlexNode,
  PlexHeading,
} from '../plex'

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
 * A record of a connection, with the end of the line its arrowhead is at: a
 * record written in the first note points at the second, and one written in the
 * second points at the first.
 */
interface Written {
  readonly side: ConnectionSide | null
  readonly end: EdgeArrow
}

const getRecords = (connection: Connection): Written[] => [
  { side: connection.firstSide, end: 'to' },
  { side: connection.secondSide, end: 'from' },
]

const isPointing = (record: Written): boolean => record.side?.marker === 'arrow'

function getEdgeLabels(records: readonly Written[]): EdgeLabel[] {
  const labels: EdgeLabel[] = []
  for (const record of records) {
    const text = record.side?.label ?? ''
    if (!text) continue
    if (!labels.some((one) => one.text === text)) {
      labels.push(isPointing(record) ? { text, end: record.end } : { text })
    }
  }
  return labels
}

function createEdge(connection: Connection, token: (path: string) => string): PlexEdge {
  const records = getRecords(connection)
  const labels = getEdgeLabels(records)
  const arrows = records.filter(isPointing).map((record) => record.end)
  return {
    from: token(connection.first),
    to: token(connection.second),
    ...(labels.length > 0 ? { labels } : {}),
    ...(arrows.length > 0 ? { arrows } : {}),
  }
}

/**
 * A node is drawn under the token its note holds, and so is each end of an
 * edge. The caller supplies the token of a note.
 *
 * Every line is drawn from one connection of the neighbourhood, whichever notes
 * it joins: the note in focus and a neighbour, or two other notes that are both
 * drawn. An arrowhead is drawn at each end whose record carries a marker, and
 * nowhere else, and the labels of its records are set by how many there are, not by the arrowheads.
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
  /** The path of every note this picture draws, the note in focus included. */
  const visiblePaths = new Set<string>([neighbourhood.focus.path])
  for (const related of neighbourhood.related) {
    visiblePaths.add(related.path)
    nodes.push({ id: token(related.path), title: related.title, relation: related.relation })
  }

  const edges = neighbourhood.connections
    .filter(({ first, second }) => visiblePaths.has(first) && visiblePaths.has(second))
    .map((connection) => createEdge(connection, token))

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

const areSidesEqual = (one: ConnectionSide | null, other: ConnectionSide | null): boolean =>
  one === null || other === null
    ? one === other
    : one.label === other.label && one.marker === other.marker

const areConnectionsEqual = (one: Connection, other: Connection): boolean =>
  one.first === other.first &&
  one.second === other.second &&
  areSidesEqual(one.firstSide, other.firstSide) &&
  areSidesEqual(one.secondSide, other.secondSide)

/**
 * Whether two neighbourhoods draw one graph: the note in focus, and every
 * note joined to it in arrival order, and every line between them with the
 * label and the arrowhead of each of its records.
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
  if (one.connections.length !== other.connections.length) return false

  return (
    one.related.every((related, index) => {
      const comparison = other.related[index]!
      return (
        related.path === comparison.path &&
        related.title === comparison.title &&
        related.type === comparison.type &&
        related.relation === comparison.relation
      )
    }) &&
    one.connections.every((connection, index) =>
      areConnectionsEqual(connection, other.connections[index]!),
    )
  )
}
