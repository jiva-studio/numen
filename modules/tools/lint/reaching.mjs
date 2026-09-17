/**
 * An import leaving its own slice says which layer it reaches.
 *
 * `../../entities/note` counts folders; `@/entities/note` names one. The two
 * resolve to the same file, so no boundary check can tell them apart — what
 * differs is whether a reader can see, without counting, that this import goes
 * down a layer rather than up one.
 *
 * Inside a slice a relative path stays relative: a slice moved whole should not
 * take a rewrite with it.
 */
import { masked } from './catches.mjs'
import { blocks } from './source.mjs'

/** The modules laid out in slices, by where their source stands. */
const LAID = ['modules/apps/desktop/editor/src/', 'modules/apps/desktop/flashcards/src/']

/** The layers cut into slices, whose folders are one deeper than the layer. */
const SLICED = new Set(['entities', 'features', 'widgets', 'pages', 'screens'])

/**
 * How many folders deep a file stands below its own slice, and nothing for a
 * file that stands in no module laid out in slices.
 *
 * A file of an unsliced layer — `shared/`, `app/` — is measured from the layer,
 * which is the whole of what it may reach relatively.
 */
export function depth(at) {
  const under = LAID.find((one) => at.startsWith(one))
  if (!under) return null
  const parts = at.slice(under.length).split('/')
  const root = SLICED.has(parts[0]) ? 2 : 1
  // A file at the module's own root stands below nothing and climbs nowhere.
  return Math.max(0, parts.length - 1 - root)
}

/**
 * Every relative specifier one file writes, with how far up it climbs.
 *
 * The mask blanks what a comment says and leaves every other character where it
 * was, so a path named in prose is nobody's business and the offsets still read
 * back onto the source.
 */
const SPECIFIER = /(?:from\s+|import\(|vi\.(?:do)?mock\()'[^'\n]*'/g

export function reached(source) {
  const over = masked(source)
  const found = []
  for (const one of over.matchAll(SPECIFIER)) {
    const opens = one.index + one[0].indexOf("'")
    const said = source.slice(opens + 1, one.index + one[0].length - 1)
    if (!said.startsWith('.')) continue
    found.push({ said, up: [...said.matchAll(/(?:^|\/)\.\.(?=\/)/g)].length })
  }
  return found
}

/** The parts of a file this rule reads: a component's are its script blocks. */
const read = (at, text) => (at.endsWith('.vue') ? blocks(text, 'script').join('\n') : text)

/**
 * Every import of a file that climbs out of its own slice by counting folders.
 *
 * A path leaving the module altogether — a fixture the schema package holds —
 * is none of this: `@/` names the window's own source and cannot say it.
 */
export function refused(at, text) {
  const own = depth(at)
  if (own === null) return []
  return reached(read(at, text))
    .filter((one) => one.up > own)
    .filter((one) => !one.said.includes('/libs/'))
    .map((one) => one.said)
}
