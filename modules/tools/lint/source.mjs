/** The interface modules' own source, and the walk that reads it. */
import { readdirSync, readFileSync, statSync } from 'node:fs'
import { dirname, extname, join, relative } from 'node:path'
import { fileURLToPath } from 'node:url'
import { modules, root } from '../modules.mjs'

export const here = dirname(fileURLToPath(import.meta.url))
export { modules, root }

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
    for (const path of walk(join(root, one.written), [])) {
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
