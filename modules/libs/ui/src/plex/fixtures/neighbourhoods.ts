/**
 * Neighbourhoods to build against, written by hand and awkward on purpose.
 *
 * None of this comes from a vault: a fixture taken from the domain is how the
 * dependency gets back in through the door marked "tests".
 */
import type { PlexEdge, PlexNeighbourhood, PlexNode, PlexSeat } from '../model'

/**
 * Focus in, everything else out — the shape every fixture below is built as.
 *
 * A sibling is the exception, and the shape that catches the most: it is
 * another of a parent's children, so its line runs from that parent and never
 * touches the focus. A fixture that drew it from the focus would be a shape no
 * producer emits, and everything swept over these would be arranged against it.
 */
function neighbourhood(
  focus: string,
  related: readonly PlexNode[],
  extra: readonly PlexEdge[] = [],
): PlexNeighbourhood {
  const nodes: PlexNode[] = [
    { id: 'focus', title: focus, seat: 'focus' },
    ...related,
  ]
  const parent = related.find((node) => node.seat === 'parent')
  const edges: PlexEdge[] = related.flatMap((node) => {
    if (node.seat === 'parent' || node.seat === 'jump') {
      return [{ from: node.id, to: 'focus' }]
    }
    if (node.seat === 'sibling') {
      return parent ? [{ from: parent.id, to: node.id }] : []
    }
    return [{ from: 'focus', to: node.id }]
  })
  return { nodes, edges: [...edges, ...extra] }
}

function run(
  seat: PlexSeat,
  count: number,
  title: (index: number) => string,
): PlexNode[] {
  return Array.from({ length: count }, (_, index) => ({
    id: `${seat}-${index}`,
    title: title(index),
    seat,
  }))
}

/** Nothing points at it and it points at nothing. A vault's first note. */
export const solitary: PlexNeighbourhood = neighbourhood('An orphan note', [])

/** The ordinary case, and the one to judge the proportions by. */
export const typical: PlexNeighbourhood = neighbourhood('Hexagonal architecture', [
  { id: 'parent-0', title: 'Software architecture', seat: 'parent' },
  { id: 'parent-1', title: 'Ports and adapters', seat: 'parent' },
  { id: 'child-0', title: 'Domain', seat: 'child' },
  { id: 'child-1', title: 'Port', seat: 'child' },
  { id: 'child-2', title: 'Adapter', seat: 'child' },
  { id: 'child-3', title: 'Composition root', seat: 'child' },
  { id: 'child-4', title: 'Use case', seat: 'child' },
  { id: 'child-5', title: 'Aggregate', seat: 'child' },
  { id: 'jump-0', title: 'Dependency inversion', seat: 'jump' },
  { id: 'jump-1', title: 'Onion architecture', seat: 'jump' },
  { id: 'jump-2', title: 'Clean architecture', seat: 'jump' },
  { id: 'sibling-0', title: 'Layered architecture', seat: 'sibling' },
  { id: 'sibling-1', title: 'Event-driven architecture', seat: 'sibling' },
])

/** Two parents, one a parent of the other: an edge inside a row. */
export const diamond: PlexNeighbourhood = neighbourhood(
  'Recursive CTE',
  [
    { id: 'parent-0', title: 'SQL', seat: 'parent' },
    { id: 'parent-1', title: 'Common table expression', seat: 'parent' },
    { id: 'child-0', title: 'Cycle detection', seat: 'child' },
    { id: 'child-1', title: 'Topological order', seat: 'child' },
  ],
  [{ from: 'parent-0', to: 'parent-1', label: 'contains' }],
)

/** Nothing above it. The top of a hierarchy has to look deliberate, not broken. */
export const root: PlexNeighbourhood = neighbourhood('Everything', [
  ...run('child', 5, (i) => ['Ideas', 'People', 'Places', 'Work', 'Reading'][i] ?? `Child ${i}`),
])

/** Nothing below it, and the plex has to stay centred anyway. */
export const leaf: PlexNeighbourhood = neighbourhood('Nothing below', [
  { id: 'parent-0', title: 'Inbox', seat: 'parent' },
])

/** One past what a side can hold, where an off-by-one in the overflow shows. */
export const crowded: PlexNeighbourhood = neighbourhood('Verbs', [
  ...run('child', 21, (i) => `Child ${i + 1}`),
])

/** Past every limit. What the overflow report is for. */
export const overcrowded: PlexNeighbourhood = neighbourhood('Everything, again', [
  ...run('parent', 9, (i) => `Parent ${i + 1}`),
  ...run('child', 200, (i) => `Child ${i + 1}`),
  ...run('jump', 40, (i) => `Jump ${i + 1}`),
])

/** Not Latin, far too long, nothing to break at, and nothing at all. */
export const awkwardLabels: PlexNeighbourhood = neighbourhood(
  'Заметка с довольно длинным названием, которое не помещается',
  [
    { id: 'parent-0', title: 'भगवद्गीता', seat: 'parent' },
    { id: 'parent-1', title: 'الفهرس', seat: 'parent' },
    { id: 'child-0', title: 'a', seat: 'child' },
    {
      id: 'child-1',
      title: 'Supercalifragilisticexpialidociousandthensomemore',
      seat: 'child',
    },
    { id: 'child-2', title: '日本語のノート', seat: 'child' },
    { id: 'child-3', title: '', seat: 'child' },
    { id: 'jump-0', title: '  leading and trailing  ', seat: 'jump' },
  ],
)

/** Every fixture, for the stories and for the tests that sweep all of them. */
export const neighbourhoods = {
  solitary,
  typical,
  diamond,
  root,
  leaf,
  crowded,
  overcrowded,
  awkwardLabels,
} as const

export type FixtureName = keyof typeof neighbourhoods
