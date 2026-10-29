/**
 * Plausible titles for an invented neighbourhood.
 *
 * Chosen from the identifier and not at random, so a layout is repeatable and a
 * screenshot of it is comparable. Realistic names matter here: a plex of
 * "Child 1, Child 2, Child 3" does not show whether labels wrap, collide or
 * read well at a glance, which is the purpose of inspecting one.
 */
import { getNow } from '../../dom'
const NAMES = [
  'Hexagonal architecture',
  'Ports and adapters',
  'Dependency inversion',
  'Composition root',
  'Domain model',
  'Value object',
  'Aggregate',
  'Repository',
  'Use case',
  'Adapter',
  'Full-text search',
  'Vector index',
  'Recursive CTE',
  'Migration',
  'Write-ahead log',
  'Cache invalidation',
  'Fingerprint',
  'Incremental update',
  'Cold rebuild',
  'Spaced repetition',
  'Review log',
  'Card identity',
  'Interval',
  'Ease factor',
  'Markdown',
  'Frontmatter',
  'Wikilink',
  'Anchor',
  'Attachment',
  'Vault',
  'Service folder',
  'Application state',
  'Structural chunking',
  'Transcript',
  'Optical recognition',
  'Page coordinates',
  'Highlighting',
  'Provenance',
  'Determinism',
  'Idempotence',
  'Back pressure',
  'Observability',
  'Race detector',
  'Fixture',
  'Humble object',
  'Functional core',
  'Imperative shell',
  'Design token',
  'Focus ring',
  'Reduced motion',
  'Right-to-left',
  'Grapheme cluster',
  'Ellipsis',
  'Viewport',
  'Hit testing',
  'Easing curve',
  'Cubic Bézier',
  'Tangent',
  'Clearance',
  'Overflow',
] as const

/** Small, stable, and enough to spread a neighbourhood across the pool. */
function hash(text: string): number {
  let value = 2166136261
  for (let i = 0; i < text.length; i++) {
    value ^= text.charCodeAt(i)
    value = Math.imul(value, 16777619)
  }
  return Math.abs(value)
}

/**
 * A name for the `index`-th node invented around `stem`. Stepping the index
 * keeps a neighbourhood free of repeats until it outgrows the pool.
 */
export const createName = (stem: string, index: number): string =>
  NAMES[(hash(stem) + index) % NAMES.length]!

/**
 * A name for a node created now, seeded by the creation time.
 *
 * A name available the instant a node exists, so no node is unnamed and no
 * typing is needed to finish a gesture. The final name of a new node is the
 * application's decision; stories use this in its place.
 */
export const nameNow = (): string => createName(String(getNow()), 0)
