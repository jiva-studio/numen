import type { PlexEdge, PlexGraph, PlexNode, PlexNodeRelation } from '../../types'
/**
 * Graphs to build against, handwritten and deliberately awkward.
 *
 * None of this comes from a vault: a fixture taken from the domain would pull
 * the dependency in through tests.
 */

function graph(
  focus: string,
  neighbors: readonly PlexNode[],
  extra: readonly PlexEdge[] = [],
): PlexGraph {
  const nodes: PlexNode[] = [{ id: 'focus', title: focus, relation: 'focus' }, ...neighbors]
  const parent = neighbors.find((node) => node.relation === 'parent')
  const edges: PlexEdge[] = neighbors.flatMap((node) => {
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
const solitary: PlexGraph = graph('An orphan note', [])

/** The ordinary case, and the one to judge the proportions by. */
const typical: PlexGraph = graph('Hexagonal architecture', [
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
const diamond: PlexGraph = graph(
  'Recursive CTE',
  [
    { id: 'parent-0', title: 'SQL', relation: 'parent' },
    { id: 'parent-1', title: 'Common table expression', relation: 'parent' },
    { id: 'child-0', title: 'Cycle detection', relation: 'child' },
    { id: 'child-1', title: 'Topological order', relation: 'child' },
  ],
  [{ from: 'parent-0', to: 'parent-1', labels: [{ text: 'contains' }] }],
)

/** Nothing above it. The top of a hierarchy has to look deliberate, not broken. */
const root: PlexGraph = graph('Everything', [
  ...run('child', 5, (i) => ['Ideas', 'People', 'Places', 'Work', 'Reading'][i] ?? `Child ${i}`),
])

/** Nothing below it, and the plex has to stay centered anyway. */
const leaf: PlexGraph = graph('Nothing below', [
  { id: 'parent-0', title: 'Inbox', relation: 'parent' },
])

/** One past what a side can hold, where an off-by-one in the overflow shows. */
const crowded: PlexGraph = graph('Verbs', [...run('child', 21, (i) => `Child ${i + 1}`)])

/** Past every limit. What the overflow report is for. */
const overcrowded: PlexGraph = graph('Everything, again', [
  ...run('parent', 9, (i) => `Parent ${i + 1}`),
  ...run('child', 200, (i) => `Child ${i + 1}`),
  ...run('jump', 40, (i) => `Jump ${i + 1}`),
])

/** Not Latin, far too long, nothing to break at, and nothing at all. */
export const awkwardLabels: PlexGraph = graph(
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
 * Every line named, and more children than a cluster shows near the focus. The lines to the far
 * ones cross the near ones, so their titles compete for space in a band the
 * near titles already occupy.
 */
const labelledRows: PlexGraph = {
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
    { from: 'focus', to: 'child-0', labels: [{ text: 'the seed shed' }] },
    { from: 'focus', to: 'child-1', labels: [{ text: 'the minutes of the meeting' }] },
    { from: 'focus', to: 'child-2', labels: [{ text: 'the rule they quoted there' }] },
    { from: 'focus', to: 'child-3', labels: [{ text: 'when a plot is taken back' }] },
    { from: 'focus', to: 'child-4', labels: [{ text: 'the chair since May' }] },
    { from: 'focus', to: 'child-5', labels: [{ text: 'in the rules' }] },
    { from: 'jump-0', to: 'focus' },
    { from: 'parent-0', to: 'sibling-0' },
    { from: 'parent-0', to: 'sibling-1' },
  ],
}

/** Every fixture, for the stories and for the tests that sweep all of them. */
export const graphs = {
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
