/**
 * The commands every plugin adds to a window's palette, read out of the
 * plugins' manifests and frontends.
 *
 * The manual lists them, and an application reaches no plugin, so the reading
 * stands here and the manual takes what it returns.
 */
import { readFile, readdir } from 'node:fs/promises'

const PLUGINS = new URL('../../plugins/', import.meta.url)

/** A source this cannot read. The caller says so in its own words. */
export class ReadError extends Error {}

const fail = (message) => {
  throw new ReadError(message)
}

/**
 * The text from the bracket at `at` to the one that closes it, and how many
 * braces open directly inside it.
 */
export const readBracketed = (source, at) => {
  let depth = 0
  let entries = 0
  let quote = null
  let i = at
  while (i < source.length) {
    const c = source[i]
    if (quote) {
      if (c === '\\') i += 1
      else if (c === quote) quote = null
      i += 1
      continue
    }
    // A bracket written in prose is not a bracket, and an apostrophe in prose
    // is not a quote: a comment is stepped over whole.
    if (c === '/' && source[i + 1] === '/') {
      const next = source.indexOf('\n', i)
      if (next < 0) return null
      i = next + 1
      continue
    }
    if (c === '/' && source[i + 1] === '*') {
      const next = source.indexOf('*/', i)
      if (next < 0) return null
      i = next + 2
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
    i += 1
  }
  return null
}

/**
 * What one entry of the words says. `at` is the file the words stand in. An
 * entry standing for another module's is followed there, which is where the
 * four tab kinds keep their own names. An entry that is a function of one name
 * is written with that name in italics, where the window puts what it names.
 */
export const readWords = async (whole, name, at, seen = new Set()) => {
  // The words a person reads are one object. What is above it says what a
  // refusal is called, under names a command has too.
  const from = whole.indexOf('export const WORDS')
  const words = from < 0 ? whole : whole.slice(from)

  const literal = words.match(new RegExp(`^\\s*${name}:\\s*'([^']+)',`, 'm'))
  if (literal) return literal[1]

  const formed = words.match(
    new RegExp(`^\\s*${name}:\\s*\\((\\w+)[^)]*\\)\\s*=>\\s*\`([^\`]*)\`,`, 'm'),
  )
  if (formed) return formed[2].replaceAll(`\${${formed[1]}}`, `*${formed[1]}*`)

  const elsewhere = words.match(new RegExp(`^\\s*${name}:\\s*([a-z]+)\\.([A-Za-z]+),`, 'm'))
  if (!elsewhere) fail(`the words say nothing under '${name}'`)
  const [, module, key] = elsewhere
  if (seen.has(module)) fail(`the words under '${name}' point at themselves`)

  // The file names where it took the module from, so a part that moved is
  // followed without this knowing where any part stands.
  const path = getWordsFile(whole, module, at)
  return readWords(await readFile(path, 'utf8'), key, path, new Set([...seen, module]))
}

/** The file a module takes one set of words in from, as `WORDS` or `WORDS as <module>`. */
const getWordsFile = (source, module, at) => {
  const named = module === 'WORDS' ? 'WORDS' : `WORDS as ${module}`
  const imported = source.match(new RegExp(`^import \\{ ${named} \\} from '([^']+)'`, 'm'))
  if (!imported) fail(`the words take nothing in as '${module}'`)
  return new URL(`${imported[1]}.ts`, at)
}

/**
 * The chord a plugin's shortcut is drawn as, such as `mod+shift+x`. The window
 * draws a shortcut that is its modifier with a letter, and Shift or not; any
 * other shortcut is refused, as the window would draw it as nothing.
 */
const getShortcutChord = (shortcut, where) => {
  const parts = shortcut.toLowerCase().split('+')
  const letter = parts.at(-1)
  const modifiers = parts.slice(0, -1)
  const known = ['mod', 'ctrl', 'control', 'cmd', 'shift']
  if (!/^[a-z]$/.test(letter) || modifiers.some((one) => !known.includes(one))) {
    fail(`${where} declares the shortcut '${shortcut}', which the window draws as nothing`)
  }
  return { letter, shift: modifiers.includes('shift') }
}

/**
 * Every plugin the window `target` loads, in the order the host lists them —
 * by id — with the commands it adds to the palette:
 *
 * - `settings`, each setting its manifest offers in the palette, drawn as the
 *   plugin's name and the setting's title;
 * - `declared`, each command its manifest declares that the palette shows,
 *   over a file where it names the files' extensions and over the window where
 *   it names none;
 * - `commands`, each command its frontend registers, over the note in front or
 *   over the window, with the chord its shortcut is drawn as.
 */
export const getPluginCommands = async (target, plugins = PLUGINS) => {
  const found = []
  for (const entry of await readdir(plugins, { withFileTypes: true })) {
    if (!entry.isDirectory()) continue
    const at = new URL(`${entry.name}/`, plugins)
    let manifest
    try {
      manifest = JSON.parse(await readFile(new URL('manifest.json', at), 'utf8'))
    } catch {
      continue
    }
    if (!(manifest.targets ?? []).includes(target)) continue
    found.push({
      id: manifest.id,
      settings: (manifest.settings ?? [])
        .filter((one) => one.palette)
        .map((one) => ({ text: `${manifest.name}: ${one.title}`, group: 'settings' })),
      declared: (manifest.commands ?? [])
        .filter((one) => !one.hiddenInPalette)
        .map((one) => ({ text: one.title, group: one.contentTypes?.length ? 'file' : 'window' })),
      commands: manifest.frontend ? await readRegisteredCommands(at, entry.name) : [],
    })
  }
  return found.sort((a, b) => (a.id < b.id ? -1 : a.id > b.id ? 1 : 0))
}

/**
 * The commands one plugin's frontend registers. Every `registerCommand` call is
 * read; one this cannot read an id and a title out of is refused.
 */
const readRegisteredCommands = async (at, name) => {
  const index = new URL('frontend/index.ts', at)
  const source = await readFile(index, 'utf8')
  const calls = [...source.matchAll(/\bregisterCommand\(\s*\{/g)]
  const commands = []
  for (const call of calls) {
    const block = readBracketed(source, call.index + call[0].length - 1)
    const id = block?.text.match(/\bid:\s*'([^']+)'/)
    const title = block?.text.match(/\btitle:\s*([A-Za-z]+)\.([A-Za-z]+)\s*,/)
    if (!id || !title) fail(`${name} registers a command this cannot read`)
    const scope = block.text.match(/\bscope:\s*'([a-z]+)'/)?.[1]
    const shortcut = block.text.match(/\bshortcut:\s*'([^']+)'/)?.[1]
    const words = getWordsFile(source, title[1], index)
    commands.push({
      id: id[1],
      text: await readWords(await readFile(words, 'utf8'), title[2], words),
      group: scope === 'note' ? 'note' : 'window',
      ...(shortcut ? { chord: getShortcutChord(shortcut, name) } : {}),
    })
  }
  return commands
}
