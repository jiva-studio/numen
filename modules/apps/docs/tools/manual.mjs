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
import { readFile, readdir, writeFile } from 'node:fs/promises'

// By path and not by name: this file is run by `node` with nothing installed,
// and a package name is a link `npm ci` makes.
import { faults, proves, stories } from '../../../tools/stories/stories.mjs'

const UI = new URL('../../desktop/editor/src/', import.meta.url)
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

/* ----------------------------------------------------------------- lists */

/**
 * The array literal one declaration holds, and how many entries stand in it.
 *
 * The entries are counted by walking the brackets, which is a reading that
 * knows nothing of what an entry says. What a regex then reads out of the same
 * text is held against this count, so a list read in part — the shape of one
 * entry changed, the rest still matching — stops the build instead of writing
 * a shorter page.
 */
const listed = (source, from) => {
  const opens = source.slice(from).match(/=>?\s*\[/)
  if (!opens) return null
  const at = from + opens.index + opens[0].length - 1

  let depth = 0
  let entries = 0
  let quote = null
  for (let i = at; i < source.length; i += 1) {
    const c = source[i]
    if (quote) {
      if (c === '\\') i += 1
      else if (c === quote) quote = null
      continue
    }
    // A bracket written in prose is not a bracket, and an apostrophe in prose
    // is not a quote: a comment is stepped over whole.
    if (c === '/' && source[i + 1] === '/') {
      i = source.indexOf('\n', i)
      if (i < 0) return null
      continue
    }
    if (c === '/' && source[i + 1] === '*') {
      i = source.indexOf('*/', i)
      if (i < 0) return null
      i += 1
      continue
    }
    if (c === "'" || c === '"' || c === '`') {
      quote = c
    } else if ('[{('.includes(c)) {
      depth += 1
      // A brace opening directly inside the array is one entry of it. Anything
      // deeper belongs to an entry already counted.
      if (c === '{' && depth === 2) entries += 1
    } else if (']})'.includes(c)) {
      depth -= 1
      if (depth === 0) return { text: source.slice(at, i + 1), entries }
    }
  }
  return null
}

/** The list one exported name is declared as. */
const declaring = (source, name, what) => {
  const at = source.indexOf(`export const ${name}`)
  const found = at < 0 ? null : listed(source, at)
  if (!found) die(what)
  return found
}

/* ------------------------------------------------------------------ keys */

/** Every chord the window carries out a command for, in the order it declares them. */
const chords = (source) => {
  const table = declaring(source, 'CHORDS', 'chords.ts no longer lists its chords')
  const found = [
    ...table.text.matchAll(
      /\{\s*command:\s*'([A-Za-z]+)',\s*letter:\s*'([a-z])',\s*shift:\s*(true|false)\s*\}/g,
    ),
  ].map(([, command, letter, shift]) => ({ command, letter, shift: shift === 'true' }))
  if (found.length === 0) die('no chords are declared in chords.ts')
  if (found.length !== table.entries) {
    die(`chords.ts lists ${table.entries} chords and this reads ${found.length}`)
  }
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
  const keys = await read(UI, 'command/chords.ts')
  const words = await read(UI, 'words.ts')

  // The two the window keeps for itself are not in that table: they put the
  // palette up, and the palette answers them.
  const palette = await read(UI, 'command/CommandPalette.vue')
  for (const letter of ['k', 'p']) {
    if (!palette.includes(`key === '${letter}'`)) {
      die(`the palette no longer answers '${letter}' itself`)
    }
  }

  const rows = [
    `| ${chordOf({ letter: 'k' })} | ${await said(words, 'find')} |`,
    `| ${chordOf({ letter: 'p' })} | Commands |`,
  ]
  for (const chord of chords(keys)) {
    rows.push(`| ${chordOf(chord)} | ${await said(words, spoken(await read(UI, 'command/commands.ts'), chord.command))} |`)
  }
  return ['| | |', '| --- | --- |', ...rows].join('\n')
}

/* -------------------------------------------------------------- commands */

/** The one list of commands, from where it opens to where it closes. */
const listing = (source) =>
  declaring(source, 'commandsOf', 'commands.ts no longer lists its commands')

/**
 * Where each row of the list begins, so that what one row says is read out of
 * the row itself and not out of a fixed number of characters after its id: a
 * row grown longer than that number is a row that stops matching.
 *
 * Every command the list holds is a command read here. One written in a shape
 * this cannot find an id in stops the build rather than dropping off the page.
 */
const rowsOf = ({ text, entries }) => {
  const found = [...text.matchAll(/\bid:\s*'([A-Za-z]+)'/g)]
  if (found.length !== entries) {
    die(`commands.ts lists ${entries} commands and this reads ${found.length}`)
  }
  return found.map((one, i) => ({
    id: one[1],
    said: text.slice(one.index, found[i + 1]?.index ?? text.length),
  }))
}

/** Which entry of the words a command is drawn with. */
const spoken = (source, command) => {
  const row = rowsOf(listing(source)).find((one) => one.id === command)
  const found = row?.said.match(/text:\s*words\.([A-Za-z]+)/)
  if (!found) die(`no row in commands.ts draws the command '${command}'`)
  return found[1]
}

/** Every command the palette offers, in the order it draws them. */
const commands = async () => {
  const source = await read(UI, 'command/commands.ts')
  const words = await read(UI, 'words.ts')
  const keys = await read(UI, 'command/chords.ts')
  const table = chords(keys)

  const declared = rowsOf(listing(source))
  if (declared.length === 0) die('no commands are declared in commands.ts')

  // Every command declared is a command the page carries. One the words or the
  // groups say nothing about stops the build rather than dropping off the page.
  const rows = declared.map(({ id, said }) => {
    const word = said.match(/text:\s*words\.([A-Za-z]+)/)
    const group = said.match(/group:\s*'(note|file|window|vault)'/)
    if (!word) die(`no words draw the command '${id}'`)
    if (!group) die(`the command '${id}' stands in no group`)
    return { id, word: word[1], group: group[1] }
  })

  const groups = [
    ['note', 'overNote'],
    ['file', 'overFile'],
    ['window', 'overWindow'],
    ['vault', 'overVault'],
  ]

  const out = []
  for (const [group, heading] of groups) {
    out.push(`### ${await said(words, heading)}`, '', '| | |', '| --- | --- |')
    for (const row of rows.filter((one) => one.group === group)) {
      const chord = table.find((one) => one.command === row.id)
      const key = chord ? chordOf(chord) : row.id === 'find' ? chordOf({ letter: 'k' }) : ''
      out.push(`| ${await said(words, row.word)} | ${key} |`)
    }
    out.push('')
  }
  return out.join('\n').trimEnd()
}

/* -------------------------------------------------------------- settings */

/**
 * The settings one Go type declares, in the order it declares them, and the
 * lines of it this could make nothing of.
 *
 * A field named and tagged is one key. A type embedded with no name of its own
 * is every key that type declares, written at this level, which is what the
 * settings file holds and so what the manual has to say. A line that is
 * neither and is still written down — one exported, tagged with something other
 * than `-` — is handed back unread, for whoever asked to refuse it: a field
 * shape nobody taught this to read is a setting quietly left out.
 */
const fieldsOf = (source, name) => {
  const at = source.search(new RegExp(`^type ${name} struct \\{$`, 'm'))
  if (at < 0) return null
  const body = source.slice(at, source.indexOf('\n}', at))

  const fields = []
  const unread = []
  let doc = []
  for (const line of body.split('\n').slice(1)) {
    const comment = line.match(/^\s*\/\/ ?(.*)$/)
    if (comment) {
      doc.push(comment[1])
      continue
    }
    const field = line.match(/^\s*([A-Z][A-Za-z0-9]*)\s+([^\s]+)\s+`json:"([^",]+)([^"]*)"`/)
    const embedded = line.match(/^\s*(\*?(?:[a-z][a-z0-9]*\.)?[A-Z][A-Za-z0-9]*)\s*$/)
    if (field) {
      const [, go, type, key] = field
      if (key !== '-') fields.push({ go, type, key, doc: doc.join(' ').trim() })
    } else if (embedded) {
      fields.push({ embedded: true, type: embedded[1], doc: doc.join(' ').trim() })
    } else if (line.trim() !== '' && !/^\s*[a-z]/.test(line) && !line.includes('json:"-"')) {
      unread.push(line.trim())
    }
    doc = []
  }
  return { fields, unread }
}

/**
 * A Go type's settings, in the order the file declares them.
 *
 * A type keeping its settings unexported writes the keys down on a mirror
 * struct beside it, so a type declaring none is read from its mirror: what a
 * person writes in the settings file is what the mirror says, whatever the
 * fields behind it come to be called. A type with neither hands back nothing
 * rather than an empty list, so that whoever asked says so out loud instead of
 * quietly losing a page of settings that still work.
 */
const structOf = (source, name) => {
  const own = fieldsOf(source, name)
  if (own === null) return null
  const mirror = fieldsOf(source, `${name[0].toLowerCase()}${name.slice(1)}File`)
  const read = own.fields.length === 0 && mirror !== null ? mirror : own
  if (read.unread.length > 0) die(`${name} declares a field this cannot read: ${read.unread[0]}`)
  return read.fields.length > 0 ? read.fields : null
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
 * Where each package's settings are declared, so the walk crosses from one to
 * the next by itself. The walk runs from `settings.Config` down, and a package
 * nobody listed here stops the build rather than the walk.
 */
const PACKAGES = {
  settings: 'adapter/settings',
  embed: 'internal/adapter/embed',
  recognition: 'internal/adapter/recognition',
  transcription: 'internal/adapter/transcription',
  proofreading: 'internal/adapter/proofreading',
  agent: 'adapter/agent',
}

const held = new Map()

/**
 * One package's Go, every file of it read as one.
 *
 * A section stands in whichever file of its package a reader put it in, and
 * what the manual is about is what the settings are rather than where they are
 * written down. Reading one named file instead makes a page of the manual
 * disappear the day a type is moved next door.
 */
const sourceOf = async (pkg) => {
  if (!held.has(pkg)) {
    const at = new URL(`${pkg}/`, GO)
    const names = (await readdir(at))
      .filter((name) => name.endsWith('.go') && !name.endsWith('_test.go'))
      .sort()
    held.set(pkg, (await Promise.all(names.map((name) => read(at, name)))).join('\n'))
  }
  return held.get(pkg)
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

/**
 * Where a field's type is declared: in this file, or in another package's.
 * Nothing for a type a person writes a value of rather than a section under.
 *
 * A package nobody listed above is where the walk would stop, and stopping
 * there costs every key beneath it, so it stops the build instead.
 */
const declaredIn = (pkg, type) => {
  const bare = type.replace(/^\*/, '')
  if (kindOf(bare) !== '' || !/^(?:[a-z][a-z0-9]*\.)?[A-Z][A-Za-z0-9]*$/.test(bare)) return null
  const elsewhere = bare.match(/^([a-z][a-z0-9]*)\.([A-Z][A-Za-z0-9]*)$/)
  if (!elsewhere) return { pkg, type: bare }
  const [, owner, name] = elsewhere
  if (!PACKAGES[owner]) die(`${pkg} names ${bare}, and no package is listed for ${owner}`)
  return { pkg: PACKAGES[owner], type: name }
}

/** Every key under one type, its own and those of the sections inside it. */
const keysOf = async (pkg, type, under, depth = 0, seen = new Set()) => {
  const source = await sourceOf(pkg)
  const keys = named(source)
  const fields = structOf(source, type)
  if (!fields) die(`${pkg} declares no settings under ${type}`)

  const out = []
  for (const field of fields) {
    // An embedded type has no key of its own: its keys are written where it is
    // embedded, at the level the settings file holds them at.
    if (field.embedded) {
      const at = declaredIn(pkg, field.type)
      if (!at) die(`${pkg} embeds ${field.type} in ${type}, which is no section`)
      const stamp = `${at.pkg}:${at.type}`
      if (seen.has(stamp)) die(`${at.type} is embedded inside itself`)
      out.push(...(await keysOf(at.pkg, at.type, under, depth, new Set([...seen, stamp]))))
      continue
    }

    const path = under ? `${under}.${field.key}` : field.key
    const at = declaredIn(pkg, field.type)
    const stamp = at && `${at.pkg}:${at.type}`
    let inside = null
    if (at && !seen.has(stamp)) {
      inside = structOf(await sourceOf(at.pkg), at.type)
      if (!inside) die(`${path} is a section, and ${at.pkg} declares no settings under ${at.type}`)
    }

    if (inside) {
      const doc = field.doc || docOf(await sourceOf(at.pkg), at.type)
      out.push({ path, depth, group: true, meaning: meaning(doc, field.go, keys), kind: '' })
      out.push(...(await keysOf(at.pkg, at.type, path, depth + 1, new Set([...seen, stamp]))))
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
  // The whole package, walked from the top: a section nobody thought to list is
  // still walked into, and a key added to one turns up here.
  const keys = await keysOf(PACKAGES.settings, 'Config', '')

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
    const under = keys.filter((one) => one.path.startsWith(`${section.path}.`))
    if (under.length === 0) die(`${section.path} is a heading with no settings under it`)
    out.push(`### \`${section.path}\``, '')
    if (section.meaning) out.push(section.meaning.replace(/^./, (c) => c.toUpperCase()), '')
    out.push('| | | |', '| --- | --- | --- |')
    for (const key of under) out.push(row(key, section.path))
    out.push('')
  }
  return out.join('\n').trimEnd()
}

/* -------------------------------------------------------------- starting */

/**
 * Every flag the application is started with. What declares one is counted
 * first and what is read out of it second, so a flag written in a form this
 * does not read is a flag the page says nothing about, and stops the build.
 */
const flags = async () => {
  const main = await read(CMD, 'main.go')
  const declared = [...main.matchAll(/\bflag\.(?:String|Bool|Float64|Int|Int64|Duration|Uint)(?:Var)?\(/g)]
  const found = [
    ...main.matchAll(
      /flag\.(?:String|Bool|Float64|Int)Var\(\s*&[^,]+,\s*"([^"]+)",\s*([^,]+),\s*(?:\n\s*)?"([^"]*)"\s*\)/g,
    ),
  ]
  if (declared.length === 0) die('no flags are declared in cmd/numen/main.go')
  if (found.length !== declared.length) {
    die(`cmd/numen/main.go declares ${declared.length} flags and this reads ${found.length}`)
  }

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

  /** Where a command stands. Anything further in is a line, not an entry. */
  const LEFT = 2

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
    // Two spaces are what holds a line apart from what it says. A line with
    // none of them that stands in under what is said carries the line above
    // on; anywhere else it is a line this cannot read, and dropping it is
    // dropping a command off the page.
    const said = line.trim().match(/^(.*?)\s{2,}(.*)$/)
    if (!said) {
      const above = blocks[holding].at(-1)
      const indent = line.length - line.trimStart().length
      if (!above || indent <= LEFT) {
        die(`cli.go says nothing about what it lists under ${holding}: ${line.trim()}`)
      }
      above[1] += ` ${line.trim()}`
      continue
    }
    blocks[holding].push([said[1], said[2]])
  }
  if (blocks.usage.length === 0) die('cli.go lists no commands')
  if (blocks.options.length === 0) die('cli.go lists no options')

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
 * Every picture in the manual is a story drawn in a palette, and a story or a
 * palette renamed is a picture that cannot be taken again. Both are held
 * against the tree here, so the renaming is caught where it happens rather
 * than the next time somebody runs the camera.
 *
 * What a story is called and what is wrong with a shot are `@numen/stories`,
 * which the landing page holds its own pictures to as well.
 */
const pictured = async () => {
  const { SHOTS } = await import('./shots.mjs')
  const { access } = await import('node:fs/promises')

  const unproved = proves()
  if (unproved) die(unproved)

  const missing = faults(SHOTS, await stories())
  if (missing.length > 0) die(missing.join('; '))

  const presets = new URL('../../../libs/core/internal/adapter/theme/presets/', import.meta.url)
  for (const shot of SHOTS) {
    if (!shot.preset) continue
    try {
      await access(new URL(`${shot.preset}.css`, presets))
    } catch {
      die(`no palette called ${shot.preset} — the picture of it cannot be taken again`)
    }
  }
}

/* ------------------------------------------------- what the rules refuse */

/**
 * The rule every list above is held to, put through sources made up here.
 *
 * A check nobody has seen fail is an assumption, and this one's whole worth is
 * the failure path, so the path is walked on every run before a page is.
 */
const REFUSES = [
  {
    what: 'a list every entry of which the pattern reads',
    source: "export const CHORDS: readonly Chord[] = [\n  { command: 'note', letter: 'n', shift: false },\n]\n",
    pattern: /\{\s*command:\s*'([A-Za-z]+)',\s*letter:\s*'([a-z])',\s*shift:\s*(true|false)\s*\}/g,
    refused: false,
  },
  {
    what: 'a chord spelled in a way the pattern does not match',
    source: "export const CHORDS: readonly Chord[] = [\n  { command: 'note', letter: 'n', shift: false },\n  { command: 'newVault', letter: 'n', shift: true },\n]\n",
    pattern: /\{\s*command:\s*'([a-z]+)',\s*letter:\s*'([a-z])',\s*shift:\s*(true|false)\s*\}/g,
    refused: true,
  },
  {
    what: 'a row of the commands carrying no id',
    source: "export const commandsOf = (): readonly Command[] => [\n  { id: 'read', group: 'note' },\n  { name: 'beside', group: 'note' },\n]\n",
    pattern: /\bid:\s*'([A-Za-z]+)'/g,
    refused: true,
  },
  {
    what: 'an apostrophe in a comment, which is prose and no quote',
    source: "export const commandsOf = (): readonly Command[] => [\n  // The vault's own folder.\n  { id: 'read', group: 'note' },\n]\n",
    pattern: /\bid:\s*'([A-Za-z]+)'/g,
    refused: false,
  },
  {
    what: 'a list that never closes',
    source: "export const CHORDS: readonly Chord[] = [\n  { command: 'note', letter: 'n', shift: false },\n",
    pattern: /\bcommand:/g,
    refused: true,
  },
]

const refuses = () => {
  if (REFUSES.length === 0) die('the rules are put through nothing')
  for (const one of REFUSES) {
    const found = listed(one.source, 0)
    const refused = !found || [...found.text.matchAll(one.pattern)].length !== found.entries
    if (refused !== one.refused) {
      die(`the rule ${one.refused ? 'lets through' : 'refuses'} ${one.what}`)
    }
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

refuses()
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
