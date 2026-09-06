/**
 * An export its own module never uses and nothing but a test draws.
 *
 * Something other than a test draws every export, or the module it stands in
 * uses it where it stands. A test is a second copy of the code's own
 * assumptions, and it holds a function up long after the window stopped asking
 * for one.
 *
 * Two things stand outside this. A name a package hands out is drawn by
 * whoever installed the package, and the barrel is where that is said; a
 * module nothing but a test draws is a fixture, read off the drawing. A story
 * is ordinary source: Storybook is built and looked at.
 */
import { existsSync, readFileSync } from 'node:fs'
import { dirname, join, normalize } from 'node:path'
import { masked } from './catches.mjs'
import { blocks, modules, root, sources } from './source.mjs'

/** The parts of a file that are code: a component's is in its script blocks. */
const codeOf = ({ at, text }) => (at.endsWith('.vue') ? blocks(text, 'script').join('\n') : text)

/** Whether a file is a test, which is the one reader that keeps nothing alive. */
const isTest = (at) => at.endsWith('.test.ts')

/** Whether a file is a story, which is drawn by the book Storybook builds. */
const isStory = (at) => at.endsWith('.stories.ts')

const DECLARED =
  /^export\s+(?:declare\s+)?(?:abstract\s+)?(?:async\s+)?(?:function\*?|class|const|let|var|type|interface|enum)\s+([A-Za-z_$][\w$]*)/gm
const CLAUSE = /^export\s+(?:type\s+)?\{([^}]*)\}(?:\s*from\s*['"]([^'"]+)['"])?/gm
const STAR = /^export\s*\*\s*(?:as\s+([A-Za-z_$][\w$]*)\s+)?from\s*['"]([^'"]+)['"]/gm
const DEFAULT = /^export\s+default\b/m
const DRAWN = /(?:^|[\n;])\s*import\s+(?:type\s+)?([^'"]*?)\s*from\s*['"]([^'"]+)['"]/g
const SIDE = /(?:^|[\n;])\s*import\s*['"]([^'"]+)['"]/g
const LATER = /\bimport\(\s*['"]([^'"]+)['"]\s*\)/g

/** Every name a module declares and exports under its own roof. */
function declares(at, code) {
  const found = at.endsWith('.vue') ? ['default'] : []
  for (const one of code.matchAll(DECLARED)) found.push(one[1])
  for (const one of code.matchAll(CLAUSE)) {
    if (one[2]) continue
    for (const part of one[1].split(',')) {
      const named = part.trim().match(/^(?:type\s+)?([A-Za-z_$][\w$]*)(?:\s+as\s+([A-Za-z_$][\w$]*))?$/)
      if (named) found.push(named[2] ?? named[1])
    }
  }
  if (DEFAULT.test(code)) found.push('default')
  return [...new Set(found)]
}

/**
 * Every name a module passes on without declaring it, as the name it hands out,
 * the name it asks its neighbour for, and where that neighbour stands. A star
 * asks for everything, and is written down as `*`.
 */
function passes(code) {
  const found = []
  for (const one of code.matchAll(CLAUSE)) {
    if (!one[2]) continue
    for (const part of one[1].split(',')) {
      const named = part.trim().match(/^(?:type\s+)?([A-Za-z_$][\w$]*)(?:\s+as\s+([A-Za-z_$][\w$]*))?$/)
      if (named) found.push({ name: named[2] ?? named[1], local: named[1], from: one[2] })
    }
  }
  for (const one of code.matchAll(STAR)) {
    found.push({ name: one[1] ?? '*', local: '*', from: one[2] })
  }
  return found
}

/** Every module a file draws from, and the names it draws out of each. */
function draws(code) {
  const found = []
  for (const one of code.matchAll(DRAWN)) {
    const names = []
    const clause = one[1]
    const braced = clause.match(/\{([\s\S]*)\}/)
    if (braced) {
      for (const part of braced[1].split(',')) {
        const named = part.trim().match(/^(?:type\s+)?([A-Za-z_$][\w$]*)(?:\s+as\s+[A-Za-z_$][\w$]*)?$/)
        if (named) names.push(named[1])
      }
    }
    const head = clause.replace(/\{[\s\S]*\}/, '').replace(/,\s*$/, '').trim()
    if (/^[A-Za-z_$][\w$]*$/.test(head)) names.push('default')
    if (/^\*\s+as\s/.test(head)) names.push('*')
    found.push({ from: one[2], names })
  }
  for (const one of code.matchAll(SIDE)) found.push({ from: one[1], names: [] })
  for (const one of code.matchAll(LATER)) found.push({ from: one[1], names: ['*'] })
  return found
}

/**
 * The file one specifier names, or null where it leaves the repository. A
 * relative path and the module's own `@/` are resolved against the files at
 * hand; the name of one of our packages resolves to that package's barrel.
 */
function pathTo(at, spec, known, barrels) {
  if (barrels.has(spec)) return barrels.get(spec)
  let base
  if (spec.startsWith('.')) base = normalize(join(dirname(at), spec))
  else if (spec.startsWith('@/')) {
    const owner = modules.find((one) => at.startsWith(`${one.written}/`))
    if (!owner) return null
    base = normalize(join(owner.written, spec.slice(2)))
  } else return null
  base = base.replace(/\.js$/, '')
  for (const one of [base, `${base}.ts`, `${base}.vue`, `${base}/index.ts`]) {
    if (known.has(one)) return one
  }
  return null
}

/**
 * What a package hands out, by package name: the source behind the one entry
 * its manifest declares. A module with no `exports` in its manifest is an
 * application, hands nothing out, and has no public surface to speak of.
 */
function barrels() {
  const found = new Map()
  for (const one of modules) {
    const manifest = JSON.parse(readFileSync(join(root, one.at, 'package.json'), 'utf8'))
    if (!manifest.exports) continue
    for (const at of [`${one.at}/src/index.ts`, `${one.at}/index.ts`]) {
      if (existsSync(join(root, at))) found.set(one.name, at)
    }
  }
  return found
}

/** Every file, with what it declares, what it passes on and what it draws. */
function held(files, barrels) {
  const known = new Set(files.map(({ at }) => at))
  const found = new Map()
  for (const one of files) {
    const code = codeOf(one)
    found.set(one.at, {
      at: one.at,
      // The code with the comments and the strings blanked out: a name only a
      // comment says is a name nothing uses.
      code: masked(code),
      own: declares(one.at, code),
      passes: passes(code).map((pass) => ({ ...pass, to: pathTo(one.at, pass.from, known, barrels) })),
      draws: draws(code).map((draw) => ({ ...draw, to: pathTo(one.at, draw.from, known, barrels) })),
    })
  }
  return found
}

/** Every name a module hands out, its own and the ones it passes on. */
function gives(at, files, seen = new Set()) {
  if (seen.has(at)) return new Set()
  seen.add(at)
  const one = files.get(at)
  if (!one) return new Set()
  const found = new Set(one.own)
  for (const pass of one.passes) {
    if (pass.local !== '*') found.add(pass.name)
    else if (pass.to) for (const name of gives(pass.to, files, seen)) found.add(name)
  }
  return found
}

/** One name of a barrel, and every declaration behind it, written into `out`. */
function publish(at, name, out, files) {
  const key = `${at}::${name}`
  if (out.has(key)) return
  out.add(key)
  const one = files.get(at)
  if (!one) return
  for (const pass of one.passes) {
    if (!pass.to) continue
    if (pass.name === name) publish(pass.to, pass.local, out, files)
    else if (pass.local === '*' && gives(pass.to, files).has(name)) publish(pass.to, name, out, files)
  }
}

/** Whether a module says a name again past the line that declares it. */
const says = (one, name) =>
  name !== 'default' && [...one.code.matchAll(new RegExp(`\\b${name}\\b`, 'g'))].length > 1

/** Who draws each name of each module, and who draws each module at all. */
function drawing(files) {
  const names = new Map()
  const readers = new Map()
  const mark = (map, key, by) => {
    if (!map.has(key)) map.set(key, new Set())
    map.get(key).add(by)
  }
  for (const one of files.values()) {
    const all = [...one.draws, ...one.passes.map((pass) => ({ to: pass.to, names: [pass.local] }))]
    for (const draw of all) {
      if (!draw.to || draw.to === one.at) continue
      mark(readers, draw.to, one.at)
      for (const name of draw.names) mark(names, `${draw.to}::${name}`, one.at)
    }
  }
  return { names, readers }
}

/**
 * Every export nothing but a test draws that its own module never uses, as the
 * file it stands in, the name, and who was holding it up.
 */
export function refused(files, surface = barrels()) {
  const all = held(files, surface)
  const { names, readers } = drawing(all)
  const open = new Set()
  for (const at of surface.values()) {
    for (const name of gives(at, all)) publish(at, name, open, all)
  }

  const wrong = []
  const written = fixtures(readers)
  for (const one of all.values()) {
    // A story is drawn by the book Storybook builds and by nothing in the tree.
    if (isTest(one.at) || isStory(one.at) || written.includes(one.at)) continue
    for (const name of one.own) {
      if (open.has(`${one.at}::${name}`)) continue
      const by = [
        ...(names.get(`${one.at}::${name}`) ?? []),
        ...(names.get(`${one.at}::*`) ?? []),
      ]
      if (by.some((reader) => !isTest(reader))) continue
      if (says(one, name)) continue
      wrong.push({ at: one.at, name, by: [...new Set(by)].sort() })
    }
  }
  return wrong
}

/**
 * Every module nothing but a test or a story draws, which is what a fixture is
 * whatever it is called and wherever it stands. Read off the drawing rather
 * than off a directory name, so no convention has to be kept in step and no
 * list of exemptions is written down.
 *
 * A component is never one. It is drawn on a screen or it is drawn nowhere,
 * and a test mounting it is not a screen. What this cannot see is a plain
 * module the window stopped asking for whole: it is drawn the way a fixture
 * is, and telling the two apart needs something no import says.
 */
function fixtures(readers) {
  const found = []
  for (const [at, drawn] of readers) {
    if (isTest(at) || at.endsWith('.vue')) continue
    if ([...drawn].every((one) => isTest(one) || isStory(one))) found.push(at)
  }
  return found.sort()
}

/** The same, over files as they stand. */
export const fixturesOf = (files, surface = barrels()) =>
  fixtures(drawing(held(files, surface)).readers)

/** Every file of the interface modules the rule reads. */
export const walked = () => sources(['.ts', '.vue'])
