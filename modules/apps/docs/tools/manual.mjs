/**
 * The four pages of the manual the application writes itself.
 *
 * A key on the keyboard page, a row on the commands page, a setting in the
 * reference and a flag on the starting page are all read out of the code that
 * answers to them, so a manual that says something the application does not do
 * fails the build. Run with `--check` it writes nothing and says which page
 * has drifted.
 *
 * What each of them means is prose, written by hand around the block. Only the
 * list itself is written here.
 */
import { readFile, writeFile } from 'node:fs/promises'

const UI = new URL('../../desktop/ui/src/', import.meta.url)
const GO = new URL('../../../libs/core/', import.meta.url)
const CMD = new URL('../../desktop/cmd/numen/', import.meta.url)
const PAGES = new URL('../src/content/docs/', import.meta.url)

/**
 * Where a written block begins and ends. A page carrying a picture is MDX, and
 * MDX has no HTML comment: what looks like one is markup it tries to read.
 */
const marks = (page) =>
  page.endsWith('.mdx')
    ? ['{/* BEGIN AUTOGEN */}', '{/* END AUTOGEN */}']
    : ['<!-- BEGIN AUTOGEN -->', '<!-- END AUTOGEN -->']

const die = (message) => {
  process.stderr.write(`manual: ${message}\n`)
  process.exit(1)
}

const read = (base, path) => readFile(new URL(path, base), 'utf8')

/* ------------------------------------------------------------------ keys */

/** Every chord the window carries out a command for, in the order it declares them. */
const chords = (keying) => {
  const found = [
    ...keying.matchAll(
      /\{\s*command:\s*'([a-z]+)',\s*letter:\s*'([a-z])',\s*shift:\s*(true|false)\s*\}/g,
    ),
  ].map(([, command, letter, shift]) => ({ command, letter, shift: shift === 'true' }))
  if (found.length === 0) die('no chords are declared in keying.ts')
  return found
}

/** The chord as it is drawn: Control and Command are one key. */
const chordOf = ({ letter, shift }) => {
  const keys = ['<kbd>Ctrl</kbd>/<kbd>⌘</kbd>']
  if (shift) keys.push('<kbd>⇧</kbd>')
  keys.push(`<kbd>${letter.toUpperCase()}</kbd>`)
  return keys.join(' ')
}

/**
 * What one entry of the words says. An entry standing for another module's is
 * followed there, which is where the four tab kinds keep their own names.
 */
const said = async (whole, name, seen = new Set()) => {
  // The words a person reads are one object. What is above it says what a
  // refusal is called, under names a command has too.
  const at = whole.indexOf('export const WORDS')
  const words = at < 0 ? whole : whole.slice(at)

  const literal = words.match(new RegExp(`^\\s*${name}:\\s*'([^']+)',`, 'm'))
  if (literal) return literal[1]

  const elsewhere = words.match(new RegExp(`^\\s*${name}:\\s*([a-z]+)\\.([A-Za-z]+),`, 'm'))
  if (!elsewhere) die(`the words say nothing under '${name}'`)
  const [, module, key] = elsewhere
  if (seen.has(module)) die(`the words under '${name}' point at themselves`)
  return said(await read(UI, `${module}/words.ts`), key, new Set([...seen, module]))
}

const keyboard = async () => {
  const keying = await read(UI, 'keying.ts')
  const words = await read(UI, 'words.ts')

  // The two the window keeps for itself are not in that table: they put the
  // field up, and the field answers them.
  const field = await read(UI, 'Field.vue')
  for (const letter of ['k', 'p']) {
    if (!field.includes(`key === '${letter}'`)) die(`the field no longer answers '${letter}' itself`)
  }

  const rows = [
    `| ${chordOf({ letter: 'k' })} | ${await said(words, 'find')} |`,
    `| ${chordOf({ letter: 'p' })} | Commands |`,
  ]
  for (const chord of chords(keying)) {
    rows.push(`| ${chordOf(chord)} | ${await said(words, spoken(await read(UI, 'commanding.ts'), chord.command))} |`)
  }
  return ['| | |', '| --- | --- |', ...rows].join('\n')
}

/* -------------------------------------------------------------- commands */

/** Which entry of the words a command is drawn with. */
const spoken = (commanding, command) => {
  const row = new RegExp(`id:\\s*'${command}',[\\s\\S]{0,140}?text:\\s*words\\.([A-Za-z]+)`)
  const found = commanding.match(row)
  if (!found) die(`no row in commanding.ts draws the command '${command}'`)
  return found[1]
}

/** Every command the palette offers, in the order it draws them. */
const commands = async () => {
  const commanding = await read(UI, 'commanding.ts')
  const words = await read(UI, 'words.ts')
  const keying = await read(UI, 'keying.ts')
  const table = chords(keying)

  const listed = commanding.slice(commanding.indexOf('export const commandsOf'))
  const rows = [
    ...listed.matchAll(
      /id:\s*'([A-Za-z]+)',\s*(?:\n\s*)?text:\s*words\.([A-Za-z]+),[\s\S]{0,220}?band:\s*'(note|window|vault)'/g,
    ),
  ].map(([, id, word, band]) => ({ id, word, band }))
  if (rows.length === 0) die('no commands are declared in commanding.ts')

  const bands = [
    ['note', 'overNote'],
    ['window', 'overWindow'],
    ['vault', 'overVault'],
  ]

  const out = []
  for (const [band, heading] of bands) {
    out.push(`### ${await said(words, heading)}`, '', '| | |', '| --- | --- |')
    for (const row of rows.filter((one) => one.band === band)) {
      const chord = table.find((one) => one.command === row.id)
      const key = chord ? chordOf(chord) : row.id === 'find' ? chordOf({ letter: 'k' }) : ''
      out.push(`| ${await said(words, row.word)} | ${key} |`)
    }
    out.push('')
  }
  return out.join('\n').trimEnd()
}

/* -------------------------------------------------------------- settings */

/** A Go type's fields, in the order the file declares them. */
const structOf = (source, name) => {
  const at = source.search(new RegExp(`^type ${name} struct \\{$`, 'm'))
  if (at < 0) return null
  const body = source.slice(at, source.indexOf('\n}', at))

  const fields = []
  let doc = []
  for (const line of body.split('\n').slice(1)) {
    const comment = line.match(/^\s*\/\/ ?(.*)$/)
    if (comment) {
      doc.push(comment[1])
      continue
    }
    const field = line.match(/^\s*([A-Z][A-Za-z0-9]*)\s+([^\s]+)\s+`json:"([^",]+)([^"]*)"`/)
    if (field) {
      const [, go, type, key] = field
      if (key !== '-') fields.push({ go, type, key, doc: doc.join(' ').trim() })
    }
    doc = []
  }
  return fields
}

/** What a Go type is called where a person reads it. */
const kindOf = (given) => {
  // A field a file may leave out is written as a pointer, which says nothing
  // about what a person puts there.
  const type = given.replace(/^\*/, '')
  if (type === 'string') return 'text'
  if (type === 'bool') return 'yes or no'
  if (['int', 'int64', 'float32', 'float64'].includes(type)) return 'a number'
  if (type === '[]string') return 'a list of words'
  return ''
}

/** The doc comment over a type, which is what a section of the file is about. */
const docOf = (source, name) => {
  const at = source.search(new RegExp(`^type ${name} `, 'm'))
  if (at < 0) return ''
  const above = source.slice(0, at).split('\n').reverse()
  const lines = []
  for (const line of above.slice(1)) {
    const comment = line.match(/^\/\/ ?(.*)$/)
    if (!comment) break
    lines.unshift(comment[1])
  }
  return lines.join(' ').replace(new RegExp(`^${name}\\s+`), '')
}

/**
 * The words a constant is written as in the file, for the comments that name
 * the constant rather than the value.
 */
const VALUES = {
  ModeSystem: '`system`',
  ModeLight: '`light`',
  ModeDark: '`dark`',
  PoolMean: '`mean`',
  PoolHead: '`head`',
}

/**
 * What a field's own comment says about it: enough sentences to say something,
 * with the Go name taken off the front and every other name written as the key
 * it is in the file.
 */
const meaning = (doc, name, keys) => {
  const said = doc
    .replace(new RegExp(`^${name}\\s+`), '')
    .replace(/^(is|are)\s+/, '')
    .replace(`${name} `, '')

  const sentences = said.match(/[^.!?]+[.!?]?/g) ?? []
  let out = ''
  for (const sentence of sentences) {
    out += sentence
    if (out.trim().length >= 45) break
  }

  const written = out
    .trim()
    .replace(/\b[A-Z][A-Za-z]+\b/g, (word) => VALUES[word] ?? (keys.get(word) ? `\`${keys.get(word)}\`` : word))
  return written.replace(/^([A-Z])(?![A-Z])/, (letter) => letter.toLowerCase())
}

/**
 * Which file declares each package's settings, so the walk crosses from one
 * to the next by itself. The whole file is walked from `settings.Config` down:
 * a section nobody listed here is still a section a key can hide in.
 */
const FILES = {
  settings: 'adapter/settings/settings.go',
  embed: 'internal/adapter/embed/config.go',
  recognition: 'internal/adapter/recognition/config.go',
  proofreading: 'internal/adapter/proofreading/config.go',
  agent: 'adapter/agent/config.go',
}

/** Every Go name in one file, against the key it is written under. */
const named = (source) => {
  const keys = new Map()
  for (const [, go, key] of source.matchAll(
    /^\s*([A-Z][A-Za-z0-9]*)\s+[^\s]+\s+`json:"([^",]+)/gm,
  )) {
    if (key !== '-') keys.set(go, key)
  }
  return keys
}

/** Where a field's type is declared: in this file, or in another package's. */
const declaredIn = (file, type) => {
  const bare = type.replace(/^\*/, '')
  const elsewhere = bare.match(/^([a-z][a-z0-9]*)\.([A-Z][A-Za-z0-9]*)$/)
  if (!elsewhere) return { file, type: bare }
  const [, pkg, name] = elsewhere
  return FILES[pkg] ? { file: FILES[pkg], type: name } : null
}

/** Every key under one type, its own and those of the sections inside it. */
const keysOf = async (file, type, under, depth = 0, seen = new Set()) => {
  const source = await read(GO, file)
  const keys = named(source)
  const fields = structOf(source, type)
  if (!fields) die(`${file} declares no type ${type}`)

  const out = []
  for (const field of fields) {
    const path = under ? `${under}.${field.key}` : field.key
    const at = declaredIn(file, field.type)
    const inside = at && structOf(await read(GO, at.file), at.type)
    const stamp = at && `${at.file}:${at.type}`

    if (inside && !seen.has(stamp)) {
      const doc = field.doc || docOf(await read(GO, at.file), at.type)
      out.push({ path, depth, group: true, meaning: meaning(doc, field.go, keys), kind: '' })
      out.push(...(await keysOf(at.file, at.type, path, depth + 1, new Set([...seen, stamp]))))
      continue
    }
    out.push({
      path,
      depth,
      group: false,
      meaning: meaning(field.doc, field.go, keys),
      kind: kindOf(field.type),
    })
  }
  return out
}

/**
 * Which groups get a heading of their own.
 *
 * The ones at the top, except that a group holding nothing but groups hands
 * the heading to each of them: `indexing` is three sections and not one.
 */
const sectioned = (keys) => {
  const under = (path, depth) =>
    keys.filter((key) => key.depth === depth && key.path.startsWith(`${path}.`))

  const out = []
  for (const key of keys.filter((one) => one.depth === 0 && one.group)) {
    const children = under(key.path, 1)
    if (children.length > 0 && children.every((child) => child.group)) out.push(...children)
    else out.push(key)
  }
  return out
}

const settings = async () => {
  // The whole file, walked from the top: a section nobody thought to list is
  // still walked into, and a key added to one turns up here.
  const keys = await keysOf(FILES.settings, 'Config', '')

  const row = (key, section) =>
    `| \`${section ? key.path.slice(section.length + 1) : key.path}\` | ${key.kind} | ${key.meaning} |`

  const out = []
  const loose = keys.filter((key) => key.depth === 0 && !key.group)
  if (loose.length > 0) {
    out.push('| | | |', '| --- | --- | --- |', ...loose.map((key) => row(key)), '')
  }

  // A section is a heading, and its keys are written under it the way they are
  // written inside it rather than as the whole path down to them.
  for (const section of sectioned(keys)) {
    out.push(`### \`${section.path}\``, '')
    if (section.meaning) out.push(section.meaning.replace(/^./, (c) => c.toUpperCase()), '')
    out.push('| | | |', '| --- | --- | --- |')
    for (const key of keys.filter((one) => one.path.startsWith(`${section.path}.`))) {
      out.push(row(key, section.path))
    }
    out.push('')
  }
  return out.join('\n').trimEnd()
}

/* -------------------------------------------------------------- starting */

const flags = async () => {
  const main = await read(CMD, 'main.go')
  const found = [
    ...main.matchAll(
      /flag\.(?:String|Bool|Float64|Int)Var\(\s*&[^,]+,\s*"([^"]+)",\s*([^,]+),\s*(?:\n\s*)?"([^"]*)"\s*\)/g,
    ),
  ]
  if (found.length === 0) die('no flags are declared in cmd/numen/main.go')

  const rows = found.map(([, name, , usage]) => {
    const said = usage.charAt(0).toUpperCase() + usage.slice(1)
    return `| \`-${name}\` | ${said}. |`
  })
  return ['| | |', '| --- | --- |', ...rows].join('\n')
}

/* ------------------------------------------------------------ the command line */

const CLI = new URL('../../../libs/core/adapter/cli/', import.meta.url)

/**
 * What the command line says it takes, out of the one string it prints when
 * asked. Two of its blocks are read: the commands, and the options standing
 * over all of them. The line above them says what the program is, which is
 * prose and is written on the page.
 */
const cli = async () => {
  const source = await read(CLI, 'cli.go')
  const usage = source.match(/const usage = `([\s\S]*?)`/)
  if (!usage) die('cli.go no longer prints a usage string')

  const blocks = { usage: [], options: [] }
  let holding = null
  for (const line of usage[1].split('\n')) {
    const opens = line.match(/^(usage|options):\s*$/)
    if (opens) {
      holding = opens[1]
      continue
    }
    if (line.trim() === '') {
      holding = null
      continue
    }
    if (!holding) continue
    const said = line.trim().match(/^(.*?)\s{2,}(.*)$/)
    if (said) blocks[holding].push([said[1], said[2]])
  }
  if (blocks.usage.length === 0) die('cli.go lists no commands')

  const rows = (of) => ['| | |', '| --- | --- |', ...of.map(([what, does]) => `| \`${what}\` | ${does} |`)]

  return [
    ...rows(blocks.usage),
    '',
    '### Over every command',
    '',
    ...rows(blocks.options),
  ].join('\n')
}

/* --------------------------------------------------------------- pictures */

/**
 * Every picture in the manual is a story, and a story renamed is a picture
 * that cannot be taken again. The names are held against the stories here, so
 * the renaming is caught where it happens rather than the next time somebody
 * runs the camera.
 */
const pictured = async () => {
  const { SHOTS } = await import('./shoot.mjs')
  const { readdir } = await import('node:fs/promises')

  /** Everywhere a story is written, which is the same list Storybook is given. */
  const roots = [
    new URL('../../../libs/ui/src/', import.meta.url),
    new URL('../../desktop/ui/src/', import.meta.url),
    new URL('../../desktop/flashcards/src/', import.meta.url),
  ]

  const told = new Set()
  for (const root of roots) {
    const files = (await readdir(root, { recursive: true })).filter((name) =>
      name.endsWith('.stories.ts'),
    )
    for (const file of files) {
      const source = await read(root, file)
      const meta = source.match(/const meta[^=]*=\s*\{[\s\S]{0,200}?title:\s*'([^']+)'/)
      if (!meta) continue
      const under = meta[1].toLowerCase().replace(/[^a-z0-9]+/g, '-')
      for (const [, name] of source.matchAll(/^export const ([A-Z][A-Za-z0-9]*)\s*:/gm)) {
        told.add(`${under}--${name.replace(/([a-z0-9])([A-Z])/g, '$1-$2').toLowerCase()}`)
      }
    }
  }

  const gone = SHOTS.filter((shot) => !told.has(shot.story))
  if (gone.length > 0) {
    die(`no story called ${gone.map((shot) => shot.story).join(', ')} — the pictures cannot be taken again`)
  }
}

/* ------------------------------------------------------------------ page */

const WRITES = [
  ['keyboard.md', keyboard],
  ['commands.mdx', commands],
  ['reference.md', settings],
  ['starting.md', flags],
  ['cli.md', cli],
]

const checking = process.argv.includes('--check')
let drifted = false

await pictured()

for (const [name, write] of WRITES) {
  const page = await read(PAGES, name)
  const [opens, closes] = marks(name)
  const begins = page.indexOf(opens)
  const ends = page.indexOf(closes)
  if (begins < 0 || ends < begins) die(`${name} has no autogen block`)

  const written = `${page.slice(0, begins + opens.length)}\n${await write()}\n${page.slice(ends)}`
  if (written === page) continue

  if (checking) {
    process.stderr.write(`manual: ${name} is not what the application says\n`)
    drifted = true
  } else {
    await writeFile(new URL(name, PAGES), written)
    process.stdout.write(`manual: ${name} was written again\n`)
  }
}

if (drifted) die('run `npm run manual`')
if (checking) process.stdout.write('manual: every page says what the application does\n')
