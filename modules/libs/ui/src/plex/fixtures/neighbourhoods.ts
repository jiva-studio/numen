/**
 * Neighbourhoods to build against, written by hand and awkward on purpose.
 *
 * None of this comes from a vault: a fixture taken from the domain is how the
 * dependency gets back in through the door marked "tests".
 */
import type { PlexEdge, PlexNeighbourhood, PlexNode, PlexRole } from '../model'

/** Focus in, everything else out — the shape every fixture below is built as. */
function neighbourhood(
  focus: string,
  related: readonly PlexNode[],
  extra: readonly PlexEdge[] = [],
): PlexNeighbourhood {
  const nodes: PlexNode[] = [
    { id: 'focus', label: focus, role: 'focus' },
    ...related,
  ]
  const edges: PlexEdge[] = related.map((node) =>
    node.role === 'parent' || node.role === 'jump'
      ? { from: node.id, to: 'focus' }
      : { from: 'focus', to: node.id },
  )
  return { nodes, edges: [...edges, ...extra] }
}

function run(
  role: PlexRole,
  count: number,
  label: (index: number) => string,
): PlexNode[] {
  return Array.from({ length: count }, (_, index) => ({
    id: `${role}-${index}`,
    label: label(index),
    role,
  }))
}

/** Nothing points at it and it points at nothing. A vault's first note. */
export const solitary: PlexNeighbourhood = neighbourhood('An orphan note', [])

/** The ordinary case, and the one to judge the proportions by. */
export const typical: PlexNeighbourhood = neighbourhood('Hexagonal architecture', [
  { id: 'parent-0', label: 'Software architecture', role: 'parent' },
  { id: 'parent-1', label: 'Ports and adapters', role: 'parent' },
  { id: 'child-0', label: 'Domain', role: 'child' },
  { id: 'child-1', label: 'Port', role: 'child' },
  { id: 'child-2', label: 'Adapter', role: 'child' },
  { id: 'child-3', label: 'Composition root', role: 'child' },
  { id: 'child-4', label: 'Use case', role: 'child' },
  { id: 'child-5', label: 'Aggregate', role: 'child' },
  { id: 'jump-0', label: 'Dependency inversion', role: 'jump' },
  { id: 'jump-1', label: 'Onion architecture', role: 'jump' },
  { id: 'jump-2', label: 'Clean architecture', role: 'jump' },
  { id: 'sibling-0', label: 'Layered architecture', role: 'sibling' },
  { id: 'sibling-1', label: 'Event-driven architecture', role: 'sibling' },
])

/** Two parents, one a parent of the other: an edge inside a row. */
export const diamond: PlexNeighbourhood = neighbourhood(
  'Recursive CTE',
  [
    { id: 'parent-0', label: 'SQL', role: 'parent' },
    { id: 'parent-1', label: 'Common table expression', role: 'parent' },
    { id: 'child-0', label: 'Cycle detection', role: 'child' },
    { id: 'child-1', label: 'Topological order', role: 'child' },
  ],
  [{ from: 'parent-0', to: 'parent-1', label: 'contains' }],
)

/** Nothing above it. The top of a hierarchy has to look deliberate, not broken. */
export const root: PlexNeighbourhood = neighbourhood('Everything', [
  ...run('child', 5, (i) => ['Ideas', 'People', 'Places', 'Work', 'Reading'][i] ?? `Child ${i}`),
])

/** Nothing below it, and the plex has to stay centred anyway. */
export const leaf: PlexNeighbourhood = neighbourhood('A passing thought', [
  { id: 'parent-0', label: 'Inbox', role: 'parent' },
])

/** Right at the line-wrap threshold, where an off-by-one shows up. */
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
    { id: 'parent-0', label: 'भगवद्गीता', role: 'parent' },
    { id: 'parent-1', label: 'الفهرس', role: 'parent' },
    { id: 'child-0', label: 'a', role: 'child' },
    {
      id: 'child-1',
      label: 'Supercalifragilisticexpialidociousandthensomemore',
      role: 'child',
    },
    { id: 'child-2', label: '日本語のノート', role: 'child' },
    { id: 'child-3', label: '', role: 'child' },
    { id: 'jump-0', label: '  leading and trailing  ', role: 'jump' },
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
