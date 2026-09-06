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
 *
 * A single-file component declares itself by its file name, which is the name
 * every template addresses it by, so `RecordingTab.vue` says "recording" by
 * standing there. Where Go has a package clause a module of TypeScript has its
 * path, so the name a test imports its neighbour by counts among the names
 * that neighbour hands it.
 *
 * What a file took from its own title answers for nothing. A `Deps` is named
 * after the module it serves and a factory after the module it is the whole
 * of, so `finding.ts` declaring `FindingDeps` and `finding` says the word
 * twice and stands on neither: a name that would be chosen anyway is the only
 * one that vouches.
 */
import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs'
import { basename, dirname, extname, join, relative } from 'node:path'
import { blocks, root, sources } from './source.mjs'

/**
 * The one Go module whose files nobody wrote. Named rather than left off: the
 * list below is read off the disk, so a module is here or it is held.
 */
const generated = new Set(['modules/libs/protocol'])

/**
 * owed are the files whose name is answered only by a word taken from that
 * name, and the list only shrinks: a rename takes its line out. Every one is a
 * module called after the doing of a thing and holding a factory called the
 * same, which is the dialect this rule was written to find.
 */
export const owed = [
  'modules/libs/core/internal/adapter/filesystem/imported_files.go',
  'modules/libs/ui/src/editor/replacing.ts',
  'modules/libs/ui/src/following/following.ts',
  'modules/apps/desktop/editor/src/note/drawing.ts',
  'modules/apps/desktop/editor/src/settings/wearing.ts',
  'modules/apps/desktop/editor/src/showing.ts',
  'modules/apps/desktop/flashcards/src/decks/scheduling.ts',
  'modules/apps/desktop/flashcards/src/session/reading.ts',
]

/**
 * Every Go module of the repository, found by its go.mod rather than written
 * down. A list written down goes stale silently, and a module left off it is a
 * rule that stops at that module's border.
 */
export function modules() {
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
  for (const one of modules()) {
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

/**
 * Whether a word reads as a verb form at all, rather than merely ending in
 * those letters. Two letters is the shortest an English verb runs to — owing,
 * doing, being — so one letter in front of the ending is a word that ends there
 * by accident: a ring is a thing and nobody rings it here.
 */
export const verbal = (word) => /^.{2,}(ing|ed)$/.test(word)

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

/** The shapes a type takes its suffix from, each named after what it belongs to. */
const ROLES = new Set(['deps', 'props', 'options', 'state', 'ref', 'handle', 'event'])

/**
 * Whether a declared name is the file's own name handed back. `FindingDeps` is
 * a `Deps` named after the module it serves and `finding` a factory named after
 * the module it is the whole of, so each is called that because the file is,
 * and a word chosen from the file name cannot then answer for it.
 *
 * A name saying anything of its own is not this: `commandsOf` says commands
 * whatever the file is called.
 */
export function echoes(stem, name) {
  const own = new Set(words(stem))
  const said = words(name)
  return said.every(
    (one, at) => own.has(one) || (at === said.length - 1 && at > 0 && ROLES.has(one)),
  )
}

/** What a test is called after what it tests, in each language. */
const BESIDE = { '.go': ['_test'], '.ts': ['.test', '.stories'], '.vue': [] }

/** The stem of a file name, and whether the file is a test. */
export function stemOf(path) {
  const kind = extname(path)
  const whole = basename(path, kind)
  for (const end of BESIDE[kind] ?? []) {
    if (whole.endsWith(end)) return { stem: whole.slice(0, -end.length), test: true }
  }
  return { stem: whole, test: false }
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
export function goDeclares(source) {
  const found = []
  for (const one of source.matchAll(TOP)) found.push(one[1] ?? one[2] ?? one[3] ?? one[4])
  for (const block of source.matchAll(GROUPED)) {
    for (const one of block[1].matchAll(IN_GROUP)) found.push(one[1])
  }
  return found
}

const EXPORTED =
  /^(?:export\s+)?(?:default\s+)?(?:declare\s+)?(?:abstract\s+)?(?:async\s+)?(?:function\*?|class|const|let|var|type|interface|enum)\s+([A-Za-z_$][\w$]*)/gm

/**
 * Every name one module of TypeScript declares. The margin is what this reads
 * too: a name bound inside a function is that function's own, and a name
 * brought in by an import is another module's.
 */
export function tsDeclares(source) {
  return [...source.matchAll(EXPORTED)].map((one) => one[1])
}

/** Every name a file's code says, with what a comment and a string say cut out. */
export function calls(source) {
  const code = source
    .replace(/\/\*[\s\S]*?\*\//g, ' ')
    .replace(/\/\/[^\n]*/g, ' ')
    .replace(/`[^`]*`/g, ' ')
    .replace(/"(?:\\.|[^"\\])*"/g, ' ')
    .replace(/'(?:\\.|[^'\\])*'/g, ' ')
  return [...code.matchAll(/[A-Za-z_$][\w$]*/g)].map((one) => one[0])
}

/** The names one file declares, in whatever it is written in. */
export function holds({ at, text }) {
  if (at.endsWith('.go')) return goDeclares(text)
  if (at.endsWith('.vue')) {
    return [stemOf(at).stem, ...blocks(text, 'script').flatMap(tsDeclares)]
  }
  return tsDeclares(text)
}

const IMPORTED = /^import\s+\(([\s\S]*?)^\)/m
const IMPORT = /^import\s+([^\n]*)$/m
const BROUGHT = /^[ \t]*(?:([A-Za-z_]\w*)\s+)?"([^"]+)"/gm

/**
 * The packages a Go file brings in and then says. `chunking.Sizes` says
 * chunking, and the word is the imported package's rather than anything this
 * file chose, so it is read whole the way a package clause is. A package
 * brought in and never qualified says nothing.
 */
export function imported(text) {
  const block = IMPORTED.exec(text) ?? IMPORT.exec(text)
  if (!block) return []
  const found = []
  for (const one of block[1].matchAll(BROUGHT)) {
    const name = one[1] ?? one[2].split('/').at(-1)
    if (new RegExp(`\\b${name}\\.`).test(text)) found.push(name)
  }
  return found
}

/**
 * What answers for a file without being a name the file chose: a Go package
 * clause, which is the folder's word and the same in every file of it, the
 * packages that file names, and a component's own name, which is what every
 * template addresses it by.
 */
export function given({ at, text }) {
  if (at.endsWith('.vue')) return [stemOf(at).stem]
  if (!at.endsWith('.go')) return []
  const clause = /^package\s+([A-Za-z_]\w*)/m.exec(text)
  return [...(clause ? [clause[1]] : []), ...imported(text)]
}

/**
 * The words of a file's name that read as a verb and that none of the names
 * the file says carries. A name the rule has nothing to say about answers with
 * an empty list.
 *
 * Only what a file chose for itself can be an echo of what the file is called,
 * so what stands for it from outside is read whole.
 */
export function refused(stem, names, given = []) {
  const verbs = words(stem).filter(verbal)
  if (verbs.length === 0) return []
  const chosen = names.filter((one) => !echoes(stem, one))
  const said = new Set([...chosen, ...given].flatMap(words))
  return verbs.filter((verb) => ![...said].some((one) => carries(verb, one)))
}

/** Every file the rule reads: the Go of the modules, and the interfaces'. */
export const allSources = () => [...goSources(), ...sources(['.ts', '.vue'])]

/**
 * What the file a test stands beside tells it. In Go that is the package
 * clause and the declarations; in TypeScript a module has no clause and is
 * imported by its path, so the path is a name of its own.
 */
const told = (one) => (one.at.endsWith('.go') ? holds(one) : [one.stem, ...holds(one)])

/**
 * Every file against the words of its name that nothing in it answers for.
 *
 * A test is read with the file it stands beside as well as itself:
 * `editing_bench_test.go` is named after `edit.go` and `ordering.test.ts`
 * after `order.ts`, so a test's stem begins with the stem of what it tests.
 */
export function named() {
  const found = allSources()
  const beside = found
    .filter(({ at }) => !stemOf(at).test)
    .map(({ at, text }) => ({ at, in: dirname(at), stem: stemOf(at).stem, text }))
  return found.map(({ at, text }) => {
    const { stem, test } = stemOf(at)
    if (!test) {
      return { at, wrong: refused(stem, holds({ at, text }), given({ at, text })) }
    }
    // A test declares little of its own and is named after what it tests, which
    // stands in another file, so nothing it says is an echo of its own name.
    const said = calls(text)
    for (const one of beside) {
      if (one.in === dirname(at) && stem.startsWith(one.stem)) said.push(...told(one))
    }
    return { at, wrong: refused(stem, [], said) }
  })
}
