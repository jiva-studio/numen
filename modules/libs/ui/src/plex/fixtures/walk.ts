/**
 * A small graph, and the rule for cutting one neighbourhood out of it.
 *
 * This is the application's job, standing in for it. None of it may move into
 * the component: the moment the plex works out who a sibling is, it knows the
 * domain. It exists because the movement can only be judged by walking a real
 * graph back and forth.
 */
import type { PlexEdge, PlexNeighbourhood, PlexNode } from '../model'

interface Named {
  readonly title: string
  readonly parents?: readonly string[]
  /** Untyped by seat: an association, not a place in the hierarchy. */
  readonly jumps?: readonly string[]
}

const GRAPH: Record<string, Named> = {
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

const parentsOf = (id: string): readonly string[] => GRAPH[id]?.parents ?? []

const childrenOf = (id: string): string[] =>
  Object.keys(GRAPH).filter((other) => parentsOf(other).includes(id))

/** The other children of my parents. Worked out here, never by the plex. */
const siblingsOf = (id: string): string[] => {
  const seen = new Set<string>()
  for (const parent of parentsOf(id)) {
    for (const child of childrenOf(parent)) {
      if (child !== id) seen.add(child)
    }
  }
  return [...seen]
}

/** Everything that points at me by association, and everything I point at. */
const jumpsOf = (id: string): string[] => {
  const mine = GRAPH[id]?.jumps ?? []
  const theirs = Object.keys(GRAPH).filter((other) =>
    (GRAPH[other]?.jumps ?? []).includes(id),
  )
  return [...new Set([...mine, ...theirs])]
}

export const walkStart = 'hexagonal'

export function neighbourhoodOf(id: string): PlexNeighbourhood {
  const focus = GRAPH[id]
  if (!focus) throw new Error(`no such node: ${id}`)

  const seat = (ids: readonly string[], seat: PlexNode['seat']): PlexNode[] =>
    ids
      .filter((other) => other !== id && GRAPH[other] !== undefined)
      .map((other) => ({ id: other, title: GRAPH[other]!.title, seat }))

  // A node reachable two ways takes one seat, first match wins. The plex
  // refuses a neighbourhood that seats the same node twice.
  const related: PlexNode[] = []
  const taken = new Set<string>([id])
  for (const group of [
    seat(parentsOf(id), 'parent'),
    seat(childrenOf(id), 'child'),
    seat(jumpsOf(id), 'jump'),
    seat(siblingsOf(id), 'sibling'),
  ]) {
    for (const node of group) {
      if (taken.has(node.id)) continue
      taken.add(node.id)
      related.push(node)
    }
  }

  return {
    nodes: [{ id, title: focus.title, seat: 'focus' }, ...related],
    edges: related.flatMap((node): PlexEdge[] => {
      if (node.seat === 'parent') return [{ from: node.id, to: id, label: 'is a' }]
      if (node.seat === 'jump') return [{ from: node.id, to: id, label: 'see also' }]
      if (node.seat === 'child') return [{ from: id, to: node.id, label: 'contains' }]

      // A sibling hangs off the parent it shares with the focus, not off the
      // focus. Working out which parent is a question about relationships, so
      // it belongs here rather than in the plex.
      const shared = parentsOf(id).find((parent) =>
        parentsOf(node.id).includes(parent),
      )
      return shared ? [{ from: shared, to: node.id, label: 'contains' }] : []
    }),
  }
}
