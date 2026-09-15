/** The repository's own source, and the walks that read it. */
import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs'
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

/** The one Go module whose files nobody wrote. */
const generated = new Set(['modules/libs/protocol'])

/** Every Go module of the repository, found by its go.mod. */
export function goModules() {
  const found = []
  for (const under of ['modules/libs', 'modules/apps']) {
    for (const one of readdirSync(join(root, under), { withFileTypes: true })) {
      if (!one.isDirectory()) continue
      const at = `${under}/${one.name}`
      if (existsSync(join(root, at, 'go.mod')) && !generated.has(at)) found.push({ at })
    }
  }
  return found
}

/** What is nobody's Go: a fixture, a generated schema, a window's own tree. */
const goSkipped = new Set(['node_modules', 'dist', 'gen', 'testdata', 'frontend'])

function goWalk(at, found) {
  for (const name of readdirSync(at)) {
    if (goSkipped.has(name)) continue
    const path = join(at, name)
    if (statSync(path).isDirectory()) goWalk(path, found)
    else if (name.endsWith('.go') && !name.endsWith('.pb.go')) found.push(path)
  }
  return found
}

/** Every hand-written Go file of the modules above, and the text in it. */
export function goSources() {
  const found = []
  for (const one of goModules()) {
    for (const path of goWalk(join(root, one.at), [])) {
      found.push({ at: relative(root, path), text: readFileSync(path, 'utf8') })
    }
  }
  return found
}

/** Where the string opened at `at` closes, or the end of its line. */
function quoted(text, at) {
  for (let i = at + 1; i < text.length; i += 1) {
    if (text[i] === '\\') i += 1
    else if (text[i] === text[at] || text[i] === '\n') return i + 1
  }
  return text.length
}

/**
 * The code of `text` with every comment and every string blanked out, each
 * line kept where it stands. A `${}` in a template literal is code, and is kept.
 */
export function code(text) {
  const out = []
  const blank = (from, to) => out.push(text.slice(from, to).replace(/[^\n]/g, ' '))
  const through = (from, closing) => {
    let depth = 0
    let i = from
    while (i < text.length) {
      const one = text[i]
      const two = text.slice(i, i + 2)
      if (two === '/*') {
        const end = text.indexOf('*/', i + 2)
        const to = end < 0 ? text.length : end + 2
        blank(i, to)
        i = to
      } else if (two === '//') {
        const end = text.indexOf('\n', i)
        const to = end < 0 ? text.length : end
        blank(i, to)
        i = to
      } else if (one === '"' || one === "'") {
        const to = quoted(text, i)
        blank(i, to)
        i = to
      } else if (one === '`') {
        i = template(i)
      } else {
        if (closing && one === '{') depth += 1
        if (closing && one === '}') {
          if (depth === 0) {
            out.push(one)
            return i + 1
          }
          depth -= 1
        }
        out.push(one)
        i += 1
      }
    }
    return i
  }
  const template = (from) => {
    out.push(' ')
    let i = from + 1
    while (i < text.length) {
      const one = text[i]
      if (one === '\\') {
        out.push('  ')
        i += 2
      } else if (one === '`') {
        out.push(' ')
        return i + 1
      } else if (text.slice(i, i + 2) === '${') {
        out.push('${')
        i = through(i + 2, '}')
      } else {
        out.push(one === '\n' ? '\n' : ' ')
        i += 1
      }
    }
    return i
  }
  through(0)
  return out.join('')
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
