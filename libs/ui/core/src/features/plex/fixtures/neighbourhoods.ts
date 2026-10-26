/**
 * Neighbourhoods to build against, written by hand and awkward on purpose.
 *
 * None of this comes from a vault: a fixture taken from the domain is how the
 * dependency gets back in through the door marked "tests".
 */
import type { PlexEdge } from '../lib/edge'
import type { PlexNeighbourhood } from '../lib/neighbourhood'
import type { PlexNode } from '../lib/node'
import type { PlexNodeRelation } from '../lib/relation'

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
  neighbours: readonly PlexNode[],
  extra: readonly PlexEdge[] = [],
): PlexNeighbourhood {
  const nodes: PlexNode[] = [{ id: 'focus', title: focus, relation: 'focus' }, ...neighbours]
  const parent = neighbours.find((node) => node.relation === 'parent')
  const edges: PlexEdge[] = neighbours.flatMap((node) => {
    if (node.relation === 'parent' || node.relation === 'jump') {
      return [{ from: node.id, to: 'focus' }]
    }
    if (node.relation === 'sibling') {
      return parent ? [{ from: parent.id, to: node.id }] : []
    }
    return [{ from: 'focus', to: node.id }]
  })
  return { nodes, edges: [...edges, ...extra] }
}

function run(
  relation: PlexNodeRelation,
  count: number,
  title: (index: number) => string,
): PlexNode[] {
  return Array.from({ length: count }, (_, index) => ({
    id: `${relation}-${index}`,
    title: title(index),
    relation,
  }))
}

/** Nothing points at it and it points at nothing. A vault's first note. */
export const solitary: PlexNeighbourhood = neighbourhood('An orphan note', [])

/** The ordinary case, and the one to judge the proportions by. */
export const typical: PlexNeighbourhood = neighbourhood('Hexagonal architecture', [
  { id: 'parent-0', title: 'Software architecture', relation: 'parent' },
  { id: 'parent-1', title: 'Ports and adapters', relation: 'parent' },
  { id: 'child-0', title: 'Domain', relation: 'child' },
  { id: 'child-1', title: 'Port', relation: 'child' },
  { id: 'child-2', title: 'Adapter', relation: 'child' },
  { id: 'child-3', title: 'Composition root', relation: 'child' },
  { id: 'child-4', title: 'Use case', relation: 'child' },
  { id: 'child-5', title: 'Aggregate', relation: 'child' },
  { id: 'jump-0', title: 'Dependency inversion', relation: 'jump' },
  { id: 'jump-1', title: 'Onion architecture', relation: 'jump' },
  { id: 'jump-2', title: 'Clean architecture', relation: 'jump' },
  { id: 'sibling-0', title: 'Layered architecture', relation: 'sibling' },
  { id: 'sibling-1', title: 'Event-driven architecture', relation: 'sibling' },
])

/** Two parents, one a parent of the other: an edge inside a row. */
export const diamond: PlexNeighbourhood = neighbourhood(
  'Recursive CTE',
  [
    { id: 'parent-0', title: 'SQL', relation: 'parent' },
    { id: 'parent-1', title: 'Common table expression', relation: 'parent' },
    { id: 'child-0', title: 'Cycle detection', relation: 'child' },
    { id: 'child-1', title: 'Topological order', relation: 'child' },
  ],
  [{ from: 'parent-0', to: 'parent-1', label: 'contains' }],
)

/** Nothing above it. The top of a hierarchy has to look deliberate, not broken. */
export const root: PlexNeighbourhood = neighbourhood('Everything', [
  ...run('child', 5, (i) => ['Ideas', 'People', 'Places', 'Work', 'Reading'][i] ?? `Child ${i}`),
])

/** Nothing below it, and the plex has to stay centred anyway. */
export const leaf: PlexNeighbourhood = neighbourhood('Nothing below', [
  { id: 'parent-0', title: 'Inbox', relation: 'parent' },
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
  'Заметка про сарай с довольно длинным названием, которое не влезает',
  [
    { id: 'parent-0', title: 'बगीचा', relation: 'parent' },
    { id: 'parent-1', title: 'الحديقة', relation: 'parent' },
    { id: 'child-0', title: 'a', relation: 'child' },
    {
      id: 'child-1',
      title: 'Supercalifragilisticexpialidociousandthensomemore',
      relation: 'child',
    },
    { id: 'child-2', title: '日本語のノート', relation: 'child' },
    { id: 'child-3', title: '', relation: 'child' },
    { id: 'jump-0', title: '  leading and trailing  ', relation: 'jump' },
  ],
)

/**
 * Every line named, and more children than one row holds. The lines to the far
 * row cross the near one, so their titles are looking for room in a band the
 * near row's titles are already standing in.
 */
export const labelledRows: PlexNeighbourhood = {
  nodes: [
    { id: 'focus', title: 'Marrowfield allotments', relation: 'focus' },
    { id: 'parent-0', title: 'Untitled note 3', relation: 'parent' },
    { id: 'child-0', title: 'Compost', relation: 'child' },
    { id: 'child-1', title: 'The question of the rota', relation: 'child' },
    { id: 'child-2', title: 'The gate is open', relation: 'child' },
    { id: 'child-3', title: 'A hedge that nobody has cut', relation: 'child' },
    { id: 'child-4', title: 'Alice Fenn, The Chair', relation: 'child' },
    { id: 'child-5', title: 'Who waters in August', relation: 'child' },
    { id: 'jump-0', title: 'xxxx', relation: 'jump' },
    { id: 'sibling-0', title: 'Untitled note 4', relation: 'sibling' },
    { id: 'sibling-1', title: 'Untitled note 5', relation: 'sibling' },
  ],
  edges: [
    { from: 'parent-0', to: 'focus' },
    { from: 'focus', to: 'child-0', label: 'the seed shed' },
    { from: 'focus', to: 'child-1', label: 'the minutes of the meeting' },
    { from: 'focus', to: 'child-2', label: 'the rule they quoted there' },
    { from: 'focus', to: 'child-3', label: 'when a plot is taken back' },
    { from: 'focus', to: 'child-4', label: 'the chair since May' },
    { from: 'focus', to: 'child-5', label: 'in the rules' },
    { from: 'jump-0', to: 'focus' },
    { from: 'parent-0', to: 'sibling-0' },
    { from: 'parent-0', to: 'sibling-1' },
  ],
}

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
  labelledRows,
} as const
