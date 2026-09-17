/**
 * The component module reaches nothing of the application.
 *
 * A component takes props it names itself and emits events carrying opaque
 * identifiers. Whoever draws it turns the application's own words into that
 * shape. The rule refuses the dependency and not the vocabulary: a component
 * that draws a vault says vault, and has no way to ask what one is.
 *
 * The dependency runs one way, so a component that imports the schema, the
 * transport or a window has turned the arrow round, and everything the module
 * exists for goes with it.
 */
import { blocks, code, sources } from './source.mjs'

/** The module the rule is about. */
export const ISOLATED = 'modules/libs/ui'

/** What nothing in it may import: the schema, the transport, and any window. */
export const REFUSED = ['@numen/protocol', '@numen/wire', '@numen/editor', '@numen/flashcards']

/** A window reached by a path rather than by its package name. */
const REFUSED_PATH = /(^|\/)apps\/(desktop|mobile)\//

/** Where a module's name stands: after `from`, and inside an `import(`. */
const NAMES = /\bfrom\b|\bimport\s*\(/g

/**
 * Every module one file names. The script of a component is read on its own,
 * and the comments are blanked so that a line written about an import is not
 * one. Blanking a comment blanks the strings with it, so where a name stands is
 * read off the blanked text and the name itself off the text as written.
 */
export function imports(text) {
  const script = text.includes('<script') ? blocks(text, 'script').join('\n') : text
  const read = code(script)
  const found = []
  for (const said of read.matchAll(NAMES)) {
    let at = said.index + said[0].length
    while (at < script.length && /[\s(]/.test(script[at])) at++
    const quote = script[at]
    if (quote !== "'" && quote !== '"') continue
    const ends = script.indexOf(quote, at + 1)
    if (ends > at) found.push(script.slice(at + 1, ends))
  }
  return found
}

/** Whether one imported name is one the component module may not reach. */
export function isRefused(named) {
  return (
    REFUSED.some((one) => named === one || named.startsWith(`${one}/`)) || REFUSED_PATH.test(named)
  )
}

/** Every import the component module makes that the rule refuses. */
export function refused(files) {
  const wrong = []
  for (const one of files) {
    if (!one.at.startsWith(`${ISOLATED}/`)) continue
    for (const named of imports(one.text)) {
      if (isRefused(named)) wrong.push(`${one.at} reaches the application: ${named}`)
    }
  }
  return wrong
}

if (process.argv[1]?.endsWith('isolated.mjs')) {
  const wrong = refused(sources(['.ts', '.vue']))
  for (const one of wrong) console.log(one)
  console.log(`\n${wrong.length} reaches`)
  process.exitCode = wrong.length > 0 ? 1 : 0
}
