/**
 * The notes related to a note, as something the plex can draw.
 *
 * This is the only file that depends on both sides. The plex receives nodes with
 * a relation and edges between them and has no concept of a note; the schema
 * has no concept of drawing.
 */
import type { Connection, ConnectionSide, LinkDirection } from '@numen/plugin-sdk'
import type {
  NoteGraph,
  NoteHeading,
  NoteType,
  RelatedNote,
  EdgeArrow,
  EdgeLabel,
  PlexEdge,
  PlexGraph,
  PlexNode,
  PlexHeading,
  ReferenceMode,
} from '../types'

/**
 * plex draws a note of a plugin-declared type with that type's icon.
 */
export function getNoteTypes(noteGraph: NoteGraph): ReadonlyMap<string, NoteType> {
  const found = new Map<string, NoteType>()
  const focus = noteGraph.focus.path
  if (focus) found.set(focus, noteGraph.focusType)
  for (const related of noteGraph.related) found.set(related.path, related.type)
  return found
}

/** The tab kind the core chose for each note of the note graph, where it reports one. */
export function getTabKinds(noteGraph: NoteGraph): ReadonlyMap<string, string> {
  const found = new Map<string, string>()
  const { focus, focusTabKind } = noteGraph
  if (focus.path && focusTabKind) found.set(focus.path, focusTabKind)
  for (const related of noteGraph.related) {
    if (related.tabKind) found.set(related.path, related.tabKind)
  }
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
  const edge: PlexEdge = { from: token(connection.first), to: token(connection.second) }
  const withLabels = labels.length > 0 ? { ...edge, labels } : edge
  return arrows.length > 0 ? { ...withLabels, arrows } : withLabels
}

/**
 * Whether the setting shows a reference: a mode shows the references written in its direction, a mutual one in
 * each of the modes that show a direction, and one with no direction in the mode that shows all.
 */
function admitReference(reference: RelatedNote, mode: ReferenceMode): boolean {
  if (mode === 'off') return false
  if (mode === 'both') return true
  return reference.direction === mode || reference.direction === 'both'
}

/** The line to a reference, with its head at the end the link points to: the note, the focus, or both. */
function createReferenceEdge(
  from: string,
  to: string,
  direction: LinkDirection | undefined,
): PlexEdge {
  const arrows: Record<LinkDirection, readonly EdgeArrow[]> = {
    outgoing: ['to'],
    incoming: ['from'],
    both: ['from', 'to'],
  }
  return direction ? { from, to, arrows: arrows[direction] } : { from, to }
}

/**
 * A node is drawn under the token its note holds, as is each end of an edge; the caller supplies a note's token.
 * Every line is drawn from one connection of the note graph, whichever notes it joins.
 * An arrowhead is drawn at each end whose record carries a marker, and nowhere else.
 * Labels are set by how many records there are, not by the arrowheads.
 */
export function convertNoteGraph(
  noteGraph: NoteGraph,
  token: (path: string) => string,
  references: ReferenceMode,
): PlexGraph {
  const focus: PlexNode = {
    id: token(noteGraph.focus.path),
    title: noteGraph.focus.title,
    relation: 'focus',
  }

  const nodes: PlexNode[] = [focus]
  /** The path of every note this picture draws, the note in focus included. */
  const visiblePaths = new Set<string>([noteGraph.focus.path])
  const shownReferences: RelatedNote[] = []
  for (const related of noteGraph.related) {
    if (related.relation === 'reference') {
      if (!admitReference(related, references)) continue
      shownReferences.push(related)
    }
    visiblePaths.add(related.path)
    nodes.push({ id: token(related.path), title: related.title, relation: related.relation })
  }

  const edges: PlexEdge[] = noteGraph.connections
    .filter(({ first, second }) => visiblePaths.has(first) && visiblePaths.has(second))
    .map((connection) => createEdge(connection, token))

  for (const reference of shownReferences) {
    edges.push(createReferenceEdge(focus.id, token(reference.path), reference.direction))
  }

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
 * Whether two note graphs draw one graph: the same note in focus, the same joined notes in arrival order,
 * and the same lines with the label and arrowhead of each record.
 * A vault that changed elsewhere returns a note graph equal to the one on screen, and an equal graph is left unchanged.
 */
export function areNoteGraphsEqual(one: NoteGraph | null, other: NoteGraph | null): boolean {
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
        related.relation === comparison.relation &&
        related.direction === comparison.direction
      )
    }) &&
    one.connections.every((connection, index) =>
      areConnectionsEqual(connection, other.connections[index]!),
    )
  )
}
