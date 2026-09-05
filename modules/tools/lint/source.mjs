/** The interface modules' own source, and the walk that reads it. */
import { readdirSync, readFileSync, statSync } from 'node:fs'
import { dirname, extname, join, relative, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

export const here = dirname(fileURLToPath(import.meta.url))
export const root = resolve(here, '../../..')

/**
 * Where each module's hand-written source stands. The list is depgraph's, and
 * for the same reason: a module left off is a rule that stops at its border.
 */
export const modules = [
  { name: '@numen/ui', at: 'modules/libs/ui/src' },
  { name: '@numen/wire', at: 'modules/libs/wire' },
  { name: '@numen/desktop-ui', at: 'modules/apps/desktop/ui/src' },
  { name: '@numen/flashcards-ui', at: 'modules/apps/desktop/flashcards/src' },
]

/** What is nobody's writing: a dependency, a build, a generated schema. */
const skipped = new Set(['node_modules', 'dist', 'gen', 'storybook-static', '.storybook'])

function walk(at, found) {
  for (const name of readdirSync(at)) {
    if (skipped.has(name)) continue
    const path = join(at, name)
    if (statSync(path).isDirectory()) walk(path, found)
    else found.push(path)
  }
  return found
}

/**
 * Every file of the modules above whose extension is one of `kinds`, as a path
 * relative to the repository and the text in it.
 */
export function sources(kinds) {
  const found = []
  for (const one of modules) {
    for (const path of walk(join(root, one.at), [])) {
      if (!kinds.includes(extname(path)) || path.endsWith('.d.ts')) continue
      found.push({ at: relative(root, path), text: readFileSync(path, 'utf8') })
    }
  }
  return found
}

/** The bodies of one kind of block in a single-file component. */
export function blocks(source, tag) {
  const found = []
  const opens = new RegExp(`<${tag}(\\s[^>]*)?>`, 'g')
  for (const one of source.matchAll(opens)) {
    const from = one.index + one[0].length
    const to = source.indexOf(`</${tag}>`, from)
    if (to > 0) found.push(source.slice(from, to))
  }
  return found
}
