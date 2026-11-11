/**
 * The five pages of the manual the application writes itself.
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
import {
  ReadError,
  getPluginCommands,
  readBracketed,
  readWords,
} from '../../../tools/ci-commands/commands.mjs'
import { hasPreset } from '../../../tools/ci-stories/presets.mjs'
import { faults, proves, stories } from '../../../tools/ci-stories/stories.mjs'
import { cell } from './cell.mjs'

const EDITOR = new URL('../../desktop/editor/src/', import.meta.url)
// The window's shared layer, which is where the words stand.
const UI = new URL('shared/ui/', EDITOR)
// The slices that declare the commands the palette lists, in the order the palette draws them, and
// the words each one names its text in. The palette's own commands name their text in the window's
// words directly.
const COMMAND_SLICES = [
  { file: 'features/note-commands/model/commands.ts', words: 'features/note-commands/words.ts' },
  { file: 'features/file-commands/model/commands.ts', words: 'features/file-commands/words.ts' },
  { file: 'features/command-palette/lib/window-commands/windowCommands.ts' },
  { file: 'features/vault-commands/model/commands.ts', words: 'features/vault-commands/words.ts' },
]
// The window the palette is drawn in, as a plugin manifest names it.
const TARGET = 'desktop.editor'
const GO = new URL('../../../libs/core/kernel/', import.meta.url)
const CMD = new URL('../../desktop/cmd/numen/', import.meta.url)
const PAGES = new URL('../src/content/docs/', import.meta.url)

/** The shared commands of the application, which plugins handle. */
const SHARED = JSON.parse(
  await readFile(new URL('../../../libs/core/protocol/commands.json', import.meta.url), 'utf8'),
)

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

/** A source the commands tool cannot read stops the build in its words. */
const refuse = (error) => {
  if (error instanceof ReadError) die(error.message)
  throw error
}

const said = async (sources, name) => {
  let unread
  for (const whole of sources) {
    try {
      return await readWords(whole, name, UI)
    } catch (error) {
      if (!(error instanceof ReadError)) throw error
      unread = error
    }
  }
  return refuse(unread)
}

/** The sources of the words the palette draws: the window's own and the palette's. */
const readWordSources = async () => [
  await read(UI, 'words.ts'),
  await read(EDITOR, 'features/command-palette/words.ts'),
]

const readPlugins = () => getPluginCommands(TARGET).catch(refuse)

/* ----------------------------------------------------------------- lists */

/**
 * The array literal one declaration holds, and how many entries stand in it.
 *
 * The entries are counted by walking the brackets, which is a reading that
 * knows nothing of what an entry says. What a regex then reads out of the same
 * text is held against this count, so a list read in part — the shape of one
 * entry changed, the rest still matching — stops the build.
 */
const listed = (source, from) => {
  const opens = source.slice(from).match(/=>?\s*\[/)
  if (!opens) return null
  return readBracketed(source, from + opens.index + opens[0].length - 1)
}

/* ------------------------------------------------------------------ keys */

/**
 * The full id of every command of the window, by the name the table gives it. The
 * names are the keys of COMMAND in ids.ts, each filed under the owner app.
 */
const readCommandIds = async () => {
  const source = await read(EDITOR, 'shared/model/command/ids.ts')
  const ids = new Map(
    [...source.matchAll(/^\s*([A-Za-z]+):\s*formatAppCommandId\('([^']+)'\)/gm)].map(
      ([, name, id]) => [name, `app:${id}`],
    ),
  )
  if (ids.size === 0) die('ids.ts no longer names its commands')
  const rows = source.matchAll(/^\s*([A-Za-z]+):\s*`\$\{COMMAND\.([A-Za-z]+)\}(#[a-z]+)`/gm)
  for (const [, name, command, argument] of rows) {
    const id = ids.get(command)
    if (id) ids.set(name, `${id}${argument}`)
  }
  return ids
}

/** The full id a name of COMMAND stands for. */
const getCommandId = (ids, name) => {
  const id = ids.get(name)
  if (!id) die(`COMMAND has no command '${name}'`)
  return id
}

/** The keys of the application, in the order commands.json declares them. */
const readKeybindings = () => {
  const found = SHARED.keybindings ?? []
  if (found.length === 0) die('commands.json declares no keybindings')
  return found.map(({ key, command, values, when }) => {
    if (Object.keys(values ?? {}).length > 0 || when) {
      die(`the keyboard page cannot draw the key '${key}', which has values or a condition`)
    }
    const parts = key.split('+')
    const letter = parts.at(-1)
    const modifiers = parts.slice(0, -1)
    if (!/^[a-z]$/.test(letter) || modifiers.some((one) => one !== 'mod' && one !== 'shift')) {
      die(`the keyboard page draws only mod and shift with a letter, and not '${key}'`)
    }
    return { command, letter, shift: modifiers.includes('shift') }
  })
}

/** The chord as it is drawn: Control and Command are one key. */
const chordOf = ({ letter, shift, alt }) => {
  const keys = ['<kbd>Ctrl</kbd>/<kbd>⌘</kbd>']
  if (alt) keys.push('<kbd>Alt</kbd>/<kbd>⌥</kbd>')
  if (shift) keys.push('<kbd>⇧</kbd>')
  keys.push(`<kbd>${letter.toUpperCase()}</kbd>`)
  return keys.join(' ')
}

const keyboard = async () => {
  const words = await readWordSources()
  const ids = await readCommandIds()
  const declared = await readCommands(ids)
  const search = getCommandId(ids, 'searchVault')
  const commands = getCommandId(ids, 'openCommands')

  // The two that put a palette up are no row of the palette: the window answers them itself.
  const text = async (command) => {
    if (command === search) return said(words, 'find')
    if (command === commands) return 'Commands'
    return said(words, spoken(declared, command))
  }

  const rows = []
  for (const one of readKeybindings()) {
    rows.push(`| ${chordOf(one)} | ${await text(one.command)} |`)
  }
  for (const plugin of await readPlugins()) {
    for (const command of plugin.commands) {
      if (command.chord) rows.push(`| ${chordOf(command.chord)} | ${command.text} |`)
    }
    for (const bound of plugin.bound) rows.push(`| ${chordOf(bound.chord)} | ${bound.text} |`)
  }
  return ['| | |', '| --- | --- |', ...rows].join('\n')
}

/* -------------------------------------------------------------- commands */

/**
 * Where each row of a slice begins, so that what one row says is read out of the row itself and
 * not out of a fixed number of characters after its id: a row grown longer than that number is a
 * row that stops matching.
 *
 * The rows of a slice stand before its handlers. Every row has an id and a group, and a count of
 * the one held against the count of the other stops the build at a row written in a shape this
 * cannot find an id in.
 */
const rowsOf = (source, ids, path) => {
  const end = source.search(/\bhandlers:/)
  const text = end < 0 ? source : source.slice(0, end)
  const found = [...text.matchAll(/\bid:\s*(?:COMMAND|ROW)\.([A-Za-z]+)/g)]
  const groups = [...text.matchAll(/^\s*group:\s*'/gm)].length
  if (found.length === 0) die(`${path} declares no commands`)
  if (found.length !== groups) {
    die(`${path} declares ${groups} groups and this reads ${found.length} commands`)
  }
  return found.map((one, i) => ({
    id: getCommandId(ids, one[1]),
    said: text.slice(one.index, found[i + 1]?.index ?? text.length),
  }))
}

/** What a slice's words name in the window's words: its own name for each entry. */
const readWordNames = (whole, path) => {
  const names = new Map(
    [...whole.matchAll(/^\s*([A-Za-z]+):\s*WORDS\.([A-Za-z]+),?$/gm)].map(([, own, entry]) => [
      own,
      entry,
    ]),
  )
  if (names.size === 0) die(`${path} no longer names its words in the window's words`)
  return names
}

/** The rows of one slice, with their text named in the window's words. */
const readSliceRows = async (slice, ids) => {
  const rows = rowsOf(await read(EDITOR, slice.file), ids, slice.file)
  if (!slice.words) return rows
  const names = readWordNames(await read(EDITOR, slice.words), slice.words)
  return rows.map(({ id, said: row }) => ({
    id,
    said: row.replace(/text:\s*[A-Z]+_WORDS\.([A-Za-z]+)/, (_whole, own) => {
      const entry = names.get(own)
      if (!entry) die(`${slice.words} has no words for '${own}'`)
      return `text: words.${entry}`
    }),
  }))
}

/** The window's own commands, read from the slices that declare them, in the order they are drawn. */
const readCommands = async (ids) => {
  const rows = []
  for (const slice of COMMAND_SLICES) rows.push(...(await readSliceRows(slice, ids)))
  return rows
}

/**
 * The window's commands over the text versions of a file: each row's group is read from the
 * version-commands slice, and its text from the words that slice declares.
 */
const readVersionCommands = async () => {
  const source = await read(EDITOR, 'features/version-commands/model/commands.ts')
  const wordsSource = await read(EDITOR, 'features/version-commands/words.ts')
  const words = wordsSource.match(/const VERSION_WORDS = \{([^}]*)\}/)?.[1]
  if (words === undefined) die('version-commands/words.ts no longer declares its words')
  const rows = [...source.matchAll(/text:\s*VERSION_WORDS\.([A-Za-z]+),\s*group:\s*'([a-z]+)'/g)]
  if (rows.length === 0) die('commands.ts declares no command')
  return rows.map(([, word, group]) => {
    const text = words.match(new RegExp(`\\b${word}:\\s*'([^']+)'`))?.[1]
    if (!text) die(`version-commands/words.ts has no words for '${word}'`)
    return { text, group }
  })
}

/** Which entry of the words a command is drawn with. */
const spoken = (declared, command) => {
  const row = declared.find((one) => one.id === command)
  const found = row?.said.match(/text:\s*words\.([A-Za-z]+)/)
  if (!found) die(`no command draws '${command}'`)
  return found[1]
}

/** That the window still draws a plugin's palette setting as its name and the setting's title. */
const checkSettingCommands = async () => {
  const source = await read(EDITOR, 'features/settings-commands/model/settingCommands.ts')
  if (!source.includes('`${owner.name}: ${setting.title}`')) {
    die(
      'settingCommands.ts no longer names a plugin setting in the palette as its plugin and title',
    )
  }
}

/**
 * The shared commands a plugin handles, in the order libs/core/protocol/commands.json lists them: over
 * the note where the command declares no required argument, and over the window where it asks for
 * one, as the window's shared commands file them.
 */
const getSharedRows = (plugins) => {
  const handled = new Set(plugins.flatMap((one) => one.handled))
  return SHARED.commands
    .filter((one) => handled.has(one.id))
    .map((one) => ({
      text: one.title,
      group: (one.arguments ?? []).some((argument) => argument.required) ? 'window' : 'note',
    }))
}

/** Every command the palette offers, in the order it draws them. */
const commands = async () => {
  const words = await readWordSources()
  const ids = await readCommandIds()
  const table = readKeybindings()

  const declared = await readCommands(ids)
  if (declared.length === 0) die('no commands are declared in the command slices')

  // Every command declared is a command the page carries. One the words or the
  // groups say nothing about stops the build.
  const own = []
  for (const { id, said: row } of declared) {
    const word = row.match(/text:\s*words\.([A-Za-z]+)/)
    const group = row.match(/group:\s*'(note|file|window|settings|vault)'/)
    if (!word) die(`no words draw the command '${id}'`)
    if (!group) die(`the command '${id}' is in no group`)
    const chord = table.find((one) => one.command === id)
    const key = chord ? chordOf(chord) : ''
    own.push({ text: await said(words, word[1]), group: group[1], key })
  }
  // A row the window draws for a command a plugin binds with values carries the key of the plugin.
  const plugins = await readPlugins()
  const bound = plugins.flatMap((one) => one.bound)
  for (const row of own) {
    const found = bound.find((one) => one.text === row.text)
    if (!row.key && found) row.key = chordOf(found.chord)
  }

  // The window's own commands come first, then what the plugins add, in the
  // order createCommands.ts hands them to the palette.
  await checkSettingCommands()
  const rows = [
    ...own,
    ...(await readVersionCommands()),
    ...plugins.flatMap((one) => one.settings),
    ...plugins.flatMap((one) => one.declared),
    ...plugins.flatMap((one) =>
      one.commands.map(({ text, group, chord }) => ({
        text,
        group,
        key: chord ? chordOf(chord) : '',
      })),
    ),
    ...getSharedRows(plugins),
  ].map((row) => ({ key: '', ...row }))

  const groups = [
    ['note', 'noteGroupTitle'],
    ['file', 'overFile'],
    ['window', 'overWindow'],
    ['settings', 'overSettings'],
    ['vault', 'overVault'],
  ]

  const out = []
  for (const [group, heading] of groups) {
    out.push(`### ${await said(words, heading)}`, '', '| | |', '| --- | --- |')
    for (const row of rows.filter((one) => one.group === group)) {
      out.push(`| ${row.text} | ${row.key} |`)
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
 * than `-` — is returned unread, for whoever asked to refuse it: a field
 * shape this was not taught to read is a setting quietly left out.
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
    const field = line.match(/^\s*([A-Z][A-Za-z0-9]*)\s+([^\s]+)\s+`json:"([^",]+)(?:,([^"]*))?"`/)
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
 * fields behind it come to be called. A type with neither returns nothing,
 * which the caller answers for.
 */
const structOf = (source, name) => {
  const own = fieldsOf(source, name)
  if (own === null) return null
  const mirror = fieldsOf(source, `${name[0].toLowerCase()}${name.slice(1)}File`)
  const readsMirror = own.fields.length === 0 && mirror !== null
  const read = readsMirror ? mirror : own
  const declared = readsMirror ? `${name[0].toLowerCase()}${name.slice(1)}File` : name
  if (read.unread.length > 0) die(`${name} declares a field this cannot read: ${read.unread[0]}`)
  if (read.fields.length > 0 && !commentAbove(source, declared).includes(SETTINGS_MARKER)) {
    die(`${declared} holds settings and has no ${SETTINGS_MARKER} line in its doc comment`)
  }
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

/**
 * The comment lines over a type. A settings type carries the line
 * `//numen:settings`: tools/checks/style/comments keeps the comments on its fields,
 * which this reads as the descriptions of the keys.
 */
const SETTINGS_MARKER = '//numen:settings'

const commentAbove = (source, name) => {
  const at = source.search(new RegExp(`^type ${name} `, 'm'))
  if (at < 0) return []
  const lines = []
  for (const line of source.slice(0, at).split('\n').reverse().slice(1)) {
    if (!line.startsWith('//')) break
    lines.unshift(line)
  }
  return lines
}

/** The doc comment over a type, which is what a section of the file is about. */
const docOf = (source, name) =>
  commentAbove(source, name)
    .filter((line) => !/^\/\/[a-z0-9]+:\S/.test(line))
    .map((line) => line.replace(/^\/\/ ?/, ''))
    .join(' ')
    .trim()
    .replace(new RegExp(`^${name}\\s+`), '')

/**
 * The words a constant is written as in the file, for the comments that name
 * the constant rather than the value.
 */
const VALUES = {
  ModeSystem: '`system`',
  ModeLight: '`light`',
  ModeDark: '`dark`',
  ProfileCustom: '`custom`',
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

  const sentences = said.match(/(?:[^.!?]|[.!?](?=\S))+[.!?]?/g) ?? []
  let out = ''
  for (const sentence of sentences) {
    out += sentence
    if (out.trim().length >= 45) break
  }

  const written = out
    .trim()
    .replace(
      /\b[A-Z][A-Za-z]+\b/g,
      (word) => VALUES[word] ?? (keys.get(word) ? `\`${keys.get(word)}\`` : word),
    )
  return written.replace(/^([A-Z])(?![A-Z])/, (letter) => letter.toLowerCase())
}

/**
 * Where each package's settings are declared, so the walk crosses from one to
 * the next by itself. The walk runs from the document down, and a package
 * no package listed here stops the build.
 */
const PACKAGES = {
  settings: 'adapter/settings',
  hardware: 'adapter/settings/hardware',
  completion: 'usecase/completion',
}

const held = new Map()

/**
 * One package's Go, every file of it read as one.
 *
 * A section is in whichever file of its package a reader put it in, so the
 * package is read whole and a type moved next door keeps its page.
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
    /^[ \t]*([A-Z][A-Za-z0-9]*)\s+[^\s]+\s+`json:"([^",]+)/gm,
  )) {
    if (key !== '-') keys.set(go, key)
  }
  return keys
}

/**
 * Where a field's type is declared: in this file, or in another package's.
 * Nothing for a type a person writes a value of. A package not listed above
 * stops the build.
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

/** What a declared setting holds, in the words of a person writing it. */
const kindOfDeclared = (one) => {
  if (one.type === 'boolean') return 'yes or no'
  if (one.type === 'number' || one.type === 'range') return 'a number'
  return 'text'
}

/** The sentence a declared setting is described by, with the values it takes. */
const meaningOfDeclared = (one) => {
  const parts = [one.description.replace(/\.$/, '')]
  if (one.optionsFrom === 'themes') parts.push('The name of an installed theme')
  else if (one.type === 'enum') {
    const values = one.options.map((option) => (typeof option === 'string' ? option : option.value))
    parts.push(`One of ${values.map((value) => `\`${value}\``).join(', ')}`)
  }
  if (one.min !== undefined && one.max !== undefined) parts.push(`From ${one.min} to ${one.max}`)
  return `${parts.join('. ')}.`.replace(/^([A-Z])(?![A-Z])/, (letter) => letter.toLowerCase())
}

const PLUGINS = new URL('../../../plugins/', import.meta.url)

/** The manifests of the plugins that declare settings, in the order of their ids. */
const readSettingPlugins = async () => {
  const found = []
  for (const entry of await readdir(PLUGINS, { withFileTypes: true })) {
    if (!entry.isDirectory()) continue
    let manifest
    try {
      manifest = JSON.parse(await read(PLUGINS, `${entry.name}/manifest.json`))
    } catch (error) {
      if (error.code === 'ENOENT') continue
      die(`plugins/${entry.name}/manifest.json cannot be read: ${error.message}`)
    }
    if ((manifest.settings ?? []).length > 0) found.push(manifest)
  }
  return found.sort((a, b) => (a.id < b.id ? -1 : a.id > b.id ? 1 : 0))
}

/**
 * The settings of the owner app. The shell declares most of them in a JSON of the format of a
 * manifest, and the reference is read from that declaration. The sections of the agent and the AI
 * route stay Go types.
 */
const settings = async () => {
  const declared = JSON.parse(await read(new URL('adapter/settings/', GO), 'app.json')).settings
  const source = await sourceOf(PACKAGES.settings)

  const sections = new Map()
  for (const one of declared) {
    const [section, ...rest] = one.key.split('.')
    const rows = sections.get(section) ?? []
    rows.push({ key: rest.join('.'), kind: kindOfDeclared(one), meaning: meaningOfDeclared(one) })
    sections.set(section, rows)
  }

  const agent = await keysOf(PACKAGES.settings, 'Agent', 'agent')
  sections.set(
    'agent',
    agent.map((key) => ({
      key: key.path.slice('agent.'.length),
      kind: key.kind,
      meaning: key.meaning,
    })),
  )
  const doc = { agent: docOf(source, 'Agent') }
  const ai = sectioned(await keysOf(PACKAGES.settings, 'AI', 'ai'))
  const aiKeys = await keysOf(PACKAGES.settings, 'AI', 'ai')
  for (const section of ai) {
    sections.set(
      section.path,
      aiKeys
        .filter((one) => one.path.startsWith(`${section.path}.`))
        .map((key) => ({
          key: key.path.slice(section.path.length + 1),
          kind: key.kind,
          meaning: key.meaning,
        })),
    )
    doc[section.path] = section.meaning
  }

  const out = [
    '| | | |',
    '| --- | --- | --- |',
    '| `v` | a number | the format version of the file. Nothing reads it yet. |',
    '| `app` |  | holds the settings of the application. |',
    '| `<plugin id>` |  | holds the settings of one plugin, next to its `enabled`. |',
    '',
  ]
  const order = ['appearance', 'agent', 'naming', 'performance', 'ai.default']
  for (const section of [
    ...order,
    ...[...sections.keys()].filter((name) => !order.includes(name)),
  ]) {
    const rows = sections.get(section)
    if (!rows || rows.length === 0) continue
    out.push(`### \`app.${section}\``, '')
    if (doc[section])
      out.push(
        doc[section].replace(/^./, (c) => c.toUpperCase()),
        '',
      )
    out.push('| | | |', '| --- | --- | --- |')
    for (const row of rows) out.push(`| \`${row.key}\` | ${row.kind} | ${row.meaning} |`)
    out.push('')
  }
  for (const manifest of await readSettingPlugins()) {
    out.push(`### \`${manifest.id}\``, '', `The settings of the plugin ${manifest.name}.`, '')
    out.push('| | | |', '| --- | --- | --- |')
    for (const one of manifest.settings) {
      out.push(`| \`${one.key}\` | ${kindOfDeclared(one)} | ${cell(meaningOfDeclared(one))} |`)
    }
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
  const declared = [
    ...main.matchAll(/\bflag\.(?:String|Bool|Float64|Int|Int64|Duration|Uint)(?:Var)?\(/g),
  ]
  const found = [
    ...main.matchAll(
      /flag\.(?:String|Bool|Float64|Int)Var\(\s*&[^,\s]+,\s*"([^"]+)",\s*([^,\s][^,]*),\s*"([^"]*)"\s*\)/g,
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

const CLI = new URL('../../../libs/core/kernel/adapter/cli/', import.meta.url)

/**
 * The text the command line prints when asked for help, put together the way
 * it puts it together: `usageHead`, the synopsis of each built-in command in
 * the order the registry lists them, then `usageTail`. A synopsis is the string
 * returned by the `Synopsis` method of a command type; the vault subcommands
 * are listed in `newVaultCommand`. Commands the plugins declare are not in the
 * source, so they are not on this page.
 */
const getUsageText = async () => {
  const constant = (source, name) => {
    const found = source.match(new RegExp('const ' + name + ' = `([^`]*)`'))
    if (!found) die(`cli.go no longer defines ${name}`)
    return found[1]
  }

  const cliSource = await read(CLI, 'cli.go')
  const head = constant(cliSource, 'usageHead')
  const tail = constant(cliSource, 'usageTail')
  if (!head.includes('usage:')) die('cli.go no longer prints a usage string')

  const names = (await readdir(CLI)).filter((n) => n.endsWith('.go') && !n.endsWith('_test.go'))
  const synopses = new Map()
  const method =
    /func \((?:\w+ )?(\w+)\) Synopsis\(\) string \{\s*return ("(?:[^"\\\n]|\\.)*"|`[^`]*`)\s*\}/g
  for (const name of names) {
    for (const [, type, literal] of (await read(CLI, name)).matchAll(method)) {
      synopses.set(type, literal.startsWith('`') ? literal.slice(1, -1) : JSON.parse(literal))
    }
  }

  const typesIn = (source, from) => {
    const start = source.indexOf(from)
    if (start === -1) die(`the cli package no longer has ${from}`)
    const end = source.indexOf('\n}', start)
    return [...source.slice(start, end).matchAll(/^\t\t(\w+)\{/gm)].map((one) => one[1])
  }

  const vault = typesIn(await read(CLI, 'vault.go'), 'func newVaultCommand')
  const builtin = []
  for (const type of typesIn(await read(CLI, 'command.go'), 'func getBuiltinCommands')) {
    if (type !== 'helpCommand') builtin.push(type)
  }
  if (!(await read(CLI, 'command.go')).includes('newVaultCommand(deps)')) {
    die('command.go no longer lists the vault commands')
  }
  const commands = [...vault, ...builtin]

  const lines = commands.map((type) => {
    const synopsis = synopses.get(type)
    if (!synopsis) die(`${type} has no synopsis the manual can read`)
    return synopsis
  })
  return head + lines.join('\n') + '\n' + tail
}

/**
 * What the command line says it takes, out of the one string it prints when
 * asked. Two of its blocks are read: the commands, and the options standing
 * over all of them. The line above them says what the program is, which is
 * prose and is written on the page.
 */
const cli = async () => {
  const usage = [null, await getUsageText()]

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
    // none of them that is in under what is said carries the line above
    // on; anywhere else it is a line this cannot read, and dropping it is
    // dropping a command off the page.
    const text = line.trim()
    const gap = text.search(/\s\s/)
    if (gap === -1) {
      const above = blocks[holding].at(-1)
      const indent = line.length - line.trimStart().length
      if (!above || indent <= LEFT) {
        die(`cli.go says nothing about what it lists under ${holding}: ${line.trim()}`)
      }
      above[1] += ` ${line.trim()}`
      continue
    }
    blocks[holding].push([text.slice(0, gap), text.slice(gap).trimStart()])
  }
  if (blocks.usage.length === 0) die('cli.go lists no commands')
  if (blocks.options.length === 0) die('cli.go lists no options')

  const rows = (of) => [
    '| | |',
    '| --- | --- |',
    ...of.map(([what, does]) => `| \`${what}\` | ${does} |`),
  ]

  return [...rows(blocks.usage), '', '### Over every command', '', ...rows(blocks.options)].join(
    '\n',
  )
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

  const unproved = proves()
  if (unproved) die(unproved)

  const missing = faults(SHOTS, await stories())
  if (missing.length > 0) die(missing.join('; '))

  for (const shot of SHOTS) {
    if (shot.preset && !hasPreset(shot.preset)) {
      die(`no palette called ${shot.preset} — the picture of it cannot be taken again`)
    }
  }
}

/* ------------------------------------------------- what the rules refuse */

/**
 * The rule every list above is held to, put through sources made up here.
 *
 * A check never seen to fail is an assumption, and this one's whole worth is
 * the failure path, so the path is walked on every run before a page is.
 */
const REFUSES = [
  {
    what: 'a list every entry of which the pattern reads',
    source:
      "export const CHORDS: readonly Chord[] = [\n  { command: COMMAND.createNote, letter: 'n', hasShift: false },\n]\n",
    pattern:
      /\{\s*command:\s*COMMAND\.([A-Za-z]+),\s*letter:\s*'([a-z])',\s*hasShift:\s*(true|false)\s*\}/g,
    refused: false,
  },
  {
    what: 'a chord spelled in a way the pattern does not match',
    source:
      "export const CHORDS: readonly Chord[] = [\n  { command: COMMAND.close, letter: 'n', shift: false },\n  { command: COMMAND.createVault, letter: 'n', shift: true },\n]\n",
    pattern:
      /\{\s*command:\s*COMMAND\.([a-z]+),\s*letter:\s*'([a-z])',\s*shift:\s*(true|false)\s*\}/g,
    refused: true,
  },
  {
    what: 'a row of the commands carrying no id',
    source:
      "export const commandsOf = (): readonly Command[] => [\n  { id: COMMAND.openNote, group: 'note' },\n  { name: 'beside', group: 'note' },\n]\n",
    pattern: /\bid:\s*COMMAND\.([A-Za-z]+)/g,
    refused: true,
  },
  {
    what: 'an apostrophe in a comment, which is prose and no quote',
    source:
      "export const commandsOf = (): readonly Command[] => [\n  // The vault's own folder.\n  { id: COMMAND.openNote, group: 'note' },\n]\n",
    pattern: /\bid:\s*COMMAND\.([A-Za-z]+)/g,
    refused: false,
  },
  {
    what: 'a list that never closes',
    source:
      "export const CHORDS: readonly Chord[] = [\n  { command: COMMAND.createNote, letter: 'n', shift: false },\n",
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
