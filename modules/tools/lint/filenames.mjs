/**
 * A file named by a gerund or a participle names something it holds.
 *
 * `owing_test.go` held the counting of vaults and said "owing" nowhere in
 * itself; `attending.go` said "attending" nowhere. A verb form on a file is
 * where a word with nothing behind it hides: no compiler reads a file name,
 * and no reader can tell from one that the word means nothing.
 *
 * So the ending is what draws the rule, and the code is what answers it. A
 * file answers with a declaration of its own — a type, a function, a constant,
 * a package. A test answers with any name its code calls, because a test is
 * named after what it tests and declares little of its own. Neither answers
 * with a comment or a string: a word only a comment says is a word the code
 * does not use.
 */
import { readdirSync, readFileSync, statSync } from 'node:fs'
import { basename, dirname, join, relative } from 'node:path'
import { root } from './source.mjs'

/**
 * Where each module's Go stands. The list is the modules', and a module left
 * off is a rule that stops at its border.
 */
export const modules = [
  { name: 'core', at: 'modules/libs/core' },
  { name: 'desktop', at: 'modules/apps/desktop' },
  { name: 'mobile', at: 'modules/apps/mobile' },
]

/** What is nobody's writing: a dependency, a fixture, a generated schema. */
const skipped = new Set(['node_modules', 'dist', 'gen', 'testdata', 'frontend'])

function walk(at, found) {
  for (const name of readdirSync(at)) {
    if (skipped.has(name)) continue
    const path = join(at, name)
    if (statSync(path).isDirectory()) walk(path, found)
    else if (name.endsWith('.go') && !name.endsWith('.pb.go')) found.push(path)
  }
  return found
}

/** Every hand-written Go file of the modules above, and the text in it. */
export function goSources() {
  const found = []
  for (const one of modules) {
    for (const path of walk(join(root, one.at), [])) {
      found.push({ at: relative(root, path), text: readFileSync(path, 'utf8') })
    }
  }
  return found
}

/** The words of one name, as a reader says them. */
export const words = (name) =>
  [...name.matchAll(/[A-Z]+(?![a-z])|[A-Z][a-z0-9]*|[a-z][a-z0-9]*/g)].map((one) =>
    one[0].toLowerCase(),
  )

/** A verb form with the tense taken off: naming and named both give nam. */
function verbStem(word) {
  let stem = word
  for (const end of ['ing', 'ed', 'es', 's']) {
    if (stem.endsWith(end) && stem.length - end.length >= 3) {
      stem = stem.slice(0, -end.length)
      if (/(.)\1$/.test(stem)) stem = stem.slice(0, -1)
      break
    }
  }
  return stem.endsWith('e') && stem.length > 3 ? stem.slice(0, -1) : stem
}

/**
 * A declared word with its plural taken off, and no more. A tense is left
 * where it stands: two verb forms of one verb are two names for the doing of
 * it, and one does not answer for the other.
 */
function nounStem(word) {
  const stem = word.endsWith('es') && word.length > 4 ? word.slice(0, -2)
    : word.endsWith('s') && word.length > 3 ? word.slice(0, -1)
    : word
  return stem.endsWith('e') && stem.length > 3 ? stem.slice(0, -1) : stem
}

/** Whether a word of a file's name is carried by a word a declaration says. */
export const carries = (said, declared) =>
  said === declared || verbStem(said) === nounStem(declared)

/** The stem of a file name, and whether the file is a test. */
export function stemOf(path) {
  const stem = basename(path, '.go')
  return stem.endsWith('_test')
    ? { stem: stem.slice(0, -5), test: true }
    : { stem, test: false }
}

const TOP =
  /^func\s+(?:\([^)]*\)\s*)?([A-Za-z_]\w*)|^type\s+([A-Za-z_]\w*)|^(?:var|const)\s+([A-Za-z_]\w*)|^package\s+([A-Za-z_]\w*)/gm
const GROUPED = /^(?:type|var|const)\s*\(\n([\s\S]*?)^\)/gm
const IN_GROUP = /^\t([A-Za-z_]\w*)/gm

/**
 * Every name one Go file declares: its package, its types, its functions and
 * methods, its constants and variables, one to a line and in a block. gofmt
 * puts a declaration of the file's own at the left margin and nothing else, so
 * a margin is all this has to read. It sees no name declared inside a function
 * body, and none a comment says.
 */
export function declares(source) {
  const found = []
  for (const one of source.matchAll(TOP)) found.push(one[1] ?? one[2] ?? one[3] ?? one[4])
  for (const block of source.matchAll(GROUPED)) {
    for (const one of block[1].matchAll(IN_GROUP)) found.push(one[1])
  }
  return found
}

/** Every name a file's code says, with what a comment and a string say cut out. */
export function calls(source) {
  const code = source
    .replace(/\/\*[\s\S]*?\*\//g, ' ')
    .replace(/\/\/[^\n]*/g, ' ')
    .replace(/`[^`]*`/g, ' ')
    .replace(/"(?:\\.|[^"\\])*"/g, ' ')
  return [...code.matchAll(/[A-Za-z_]\w*/g)].map((one) => one[0])
}

/**
 * The words of a file's name that read as a verb and that none of the names
 * the file says carries. A name the rule has nothing to say about answers with
 * an empty list.
 */
export function refused(stem, names) {
  const verbs = words(stem).filter((one) => /(ing|ed)$/.test(one))
  if (verbs.length === 0) return []
  const said = new Set(names.flatMap(words))
  return verbs.filter((verb) => ![...said].some((one) => carries(verb, one)))
}

/**
 * Every Go file against the words of its name that nothing in it answers for.
 *
 * A test is read with the file it stands beside as well as itself:
 * `editing_bench_test.go` is named after `edit.go`, so a test's stem begins
 * with the stem of what it tests.
 */
export function named() {
  const sources = goSources()
  const beside = sources
    .filter(({ at }) => !stemOf(at).test)
    .map(({ at, text }) => ({ in: dirname(at), stem: stemOf(at).stem, text }))
  return sources.map(({ at, text }) => {
    const { stem, test } = stemOf(at)
    if (!test) return { at, wrong: refused(stem, declares(text)) }
    const said = calls(text)
    for (const one of beside) {
      if (one.in === dirname(at) && stem.startsWith(one.stem)) said.push(...declares(one.text))
    }
    return { at, wrong: refused(stem, said) }
  })
}
