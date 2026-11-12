import type { PlexEdge, PlexNeighborhood, PlexNode } from '../../types'
/**
 * A small graph, and the rule for extracting one neighborhood from it.
 *
 * This is the application's job, stubbed. None of it may move into the component:
 * once the plex works out who a sibling is, it depends on the domain. It exists because
 * movement can only be judged by walking a real graph back and forth.
 */

interface GraphNode {
  readonly title: string
  readonly parents?: readonly string[]
  readonly jumps?: readonly string[]
}

const GRAPH: Record<string, GraphNode> = {
  architecture: { title: 'Architecture' },
  storage: { title: 'Storage', parents: ['architecture'] },
  interface: { title: 'Interface', parents: ['architecture'] },
  hexagonal: {
    title: 'Hexagonal architecture',
    parents: ['architecture'],
    jumps: ['dependency-inversion'],
  },

  sqlite: { title: 'SQLite', parents: ['storage'], jumps: ['index'] },
  index: { title: 'The index', parents: ['storage'] },
  migration: { title: 'Migration', parents: ['storage', 'sqlite'] },
  fts: { title: 'Full-text search', parents: ['index', 'sqlite'] },
  vectors: { title: 'Vector search', parents: ['index'] },

  plex: { title: 'The plex', parents: ['interface'], jumps: ['hexagonal'] },
  editor: { title: 'Editor', parents: ['interface'] },
  review: { title: 'Review session', parents: ['interface'] },

  domain: { title: 'Domain', parents: ['hexagonal'] },
  port: { title: 'Port', parents: ['hexagonal'] },
  adapter: { title: 'Adapter', parents: ['hexagonal', 'port'] },
  'dependency-inversion': { title: 'Dependency inversion' },
}

const getParents = (id: string): readonly string[] => GRAPH[id]?.parents ?? []

const getChildren = (id: string): string[] =>
  Object.keys(GRAPH).filter((other) => getParents(other).includes(id))

const getSiblings = (id: string): string[] => {
  const observed = new Set<string>()
  for (const parent of getParents(id)) {
    for (const child of getChildren(parent)) {
      if (child !== id) observed.add(child)
    }
  }
  return [...observed]
}

const getJumps = (id: string): string[] => {
  const mine = GRAPH[id]?.jumps ?? []
  const theirs = Object.keys(GRAPH).filter((other) => (GRAPH[other]?.jumps ?? []).includes(id))
  return [...new Set([...mine, ...theirs])]
}

export const walkStart = 'hexagonal'

export function createNeighborhood(id: string): PlexNeighborhood {
  const focus = GRAPH[id]
  if (!focus) throw new Error(`no such node: ${id}`)

  const createRelatedNodes = (ids: readonly string[], relation: PlexNode['relation']): PlexNode[] =>
    ids
      .filter((other) => other !== id && GRAPH[other] !== undefined)
      .map((other) => ({ id: other, title: GRAPH[other]!.title, relation }))

  // A node reachable two ways takes one relation, first match wins. The plex rejects
  // a neighborhood that contains the same node twice.
  const related: PlexNode[] = []
  const visited = new Set<string>([id])
  for (const group of [
    createRelatedNodes(getParents(id), 'parent'),
    createRelatedNodes(getChildren(id), 'child'),
    createRelatedNodes(getJumps(id), 'jump'),
    createRelatedNodes(getSiblings(id), 'sibling'),
  ]) {
    for (const node of group) {
      if (visited.has(node.id)) continue
      visited.add(node.id)
      related.push(node)
    }
  }

  return {
    nodes: [{ id, title: focus.title, relation: 'focus' }, ...related],
    edges: related.flatMap((node): PlexEdge[] => {
      if (node.relation === 'parent') return [{ from: node.id, to: id, labels: [{ text: 'is a' }] }]
      if (node.relation === 'jump')
        return [{ from: node.id, to: id, labels: [{ text: 'see also' }] }]
      if (node.relation === 'child')
        return [{ from: id, to: node.id, labels: [{ text: 'contains' }] }]

      // A sibling hangs off the parent it shares with the focus, not off the
      // focus. Working out which parent is a question about relationships, so
      // it belongs here and not in the plex.
      const shared = getParents(id).find((parent) => getParents(node.id).includes(parent))
      return shared ? [{ from: shared, to: node.id, labels: [{ text: 'contains' }] }] : []
    }),
  }
}
