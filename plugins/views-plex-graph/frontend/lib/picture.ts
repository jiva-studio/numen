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
  NoteRelation,
  NoteType,
  RelatedNote,
  EdgeArrow,
  EdgeLabel,
  PlexEdge,
  PlexGraph,
  PlexNode,
  PlexHeading,
  PlexName,
  PlexNames,
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

function admitReference(reference: RelatedNote, mode: ReferenceMode): boolean {
  if (mode === 'off') return false
  if (mode === 'both') return true
  return reference.direction === mode || reference.direction === 'both'
}

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
 * A reference is drawn when the mode shows its direction (a mutual one in each mode that shows a direction, one with no direction in "both" only),
 * as a line from the focus with its head at the end the link points to, and no label.
 */
export function convertNoteGraph(
  noteGraph: NoteGraph,
  token: (path: string) => string,
  references: ReferenceMode = 'both',
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
    if (related.relatedTo) continue
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

/** The order the names of a note stand in, parents first. A sibling is never a name. */
const NAME_ORDER: readonly NoteRelation[] = ['parent', 'child', 'jump', 'reference']

/**
 * The names of the notes related to each related note: its parents over it, and its children, jumps and references under it,
 * cut to `limit` in that order. A name has the id `token` gives its path. A note with no names has no entry.
 */
export function getNames(
  noteGraph: NoteGraph,
  limit: number,
  token: (path: string) => string,
): ReadonlyMap<string, PlexNames> {
  const owners = new Map<string, RelatedNote[]>()
  for (const item of noteGraph.related) {
    if (!item.relatedTo || !NAME_ORDER.includes(item.relation)) continue
    const owned = owners.get(item.relatedTo) ?? []
    owned.push(item)
    owners.set(item.relatedTo, owned)
  }

  const found = new Map<string, PlexNames>()
  for (const [owner, items] of owners) {
    const ordered = NAME_ORDER.flatMap((relation) => items.filter((item) => item.relation === relation))
    const kept = ordered.slice(0, limit)
    const toName = (item: RelatedNote): PlexName => ({ id: token(item.path), title: item.title })
    found.set(token(owner), {
      above: kept.filter((item) => item.relation === 'parent').map(toName),
      below: kept.filter((item) => item.relation !== 'parent').map(toName),
    })
  }
  return found
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
        related.direction === comparison.direction &&
        related.relatedTo === comparison.relatedTo
      )
    }) &&
    one.connections.every((connection, index) =>
      areConnectionsEqual(connection, other.connections[index]!),
    )
  )
}

const NAME_PREFIX = 'name:'

/**
 * The id a name is handed to the plex under: the path of its note, in a word of its own, so that
 * choosing it reaches the note whether or not that note is drawn as a node.
 */
export const getNameToken = (path: string): string => `${NAME_PREFIX}${path}`

/** The path of the note a name stands for, and empty for an id that is not a name. */
export const getNamePath = (id: string): string =>
  id.startsWith(NAME_PREFIX) ? id.slice(NAME_PREFIX.length) : ''
