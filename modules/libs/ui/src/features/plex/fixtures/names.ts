/**
 * Plausible titles for an invented neighbourhood.
 *
 * Picked from the identifier rather than at random, so a walk is repeatable
 * and a screenshot of it means something. Real-looking names matter here: a
 * plex of "Child 1, Child 2, Child 3" tells a reader nothing about whether the
 * labels wrap, collide or read well at a glance, which is the whole reason to
 * look at one.
 */
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
export const nameFor = (stem: string, index: number): string =>
  NAMES[(hash(stem) + index) % NAMES.length]!

/**
 * A name for a node made just now, seeded by the moment it was made.
 *
 * Something to call a node the instant it exists, so nothing is ever
 * nameless and nothing has to be typed to finish a gesture. What a new node
 * should really be called is the application's to decide; this is what a
 * story uses in its place.
 */
export const nameNow = (): string => nameFor(String(Date.now()), 0)
