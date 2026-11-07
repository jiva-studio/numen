/**
 * The commands every plugin adds to a window's palette, read out of the
 * plugins' manifests and frontends.
 *
 * The manual lists them, and an application reaches no plugin, so the reading
 * stands here and the manual takes what it returns.
 */
import { readFile } from 'node:fs/promises'
import { fileURLToPath } from 'node:url'
import { ManifestError, readPluginManifests } from '../shared/manifests.mjs'
import { listRegistrations } from '../shared/registrations.mjs'

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

/** The palette group of a command by its when condition, as the window's getWhenGroup reads it. */
const getWhenGroup = (when) => {
  if (/\btarget\.isNote\b/.test(when)) return 'note'
  return /\btarget\.\w/.test(when) ? 'file' : 'window'
}

/** Whether a when condition names a text version, so the command lists one item per version. */
const isOverVersion = (when) => /\bversion\.\w/.test(when)

/**
 * Every plugin the window `target` loads, in the order the host lists them —
 * by id — with the commands it adds to the palette:
 *
 * - `settings`, each setting its manifest offers in the palette, drawn as the
 *   plugin's name and the setting's title;
 * - `declared`, each command its manifest declares that the palette shows,
 *   leaving out the ones whose condition names a text version;
 * - `commands`, each command its manifest declares that its frontend registers a
 *   handler for, with the chord its shortcut is drawn as.
 *
 * A command is in the group the window files it under by its condition: over the note where it
 * reads target.isNote, over a file where it reads another field of target, and over the window
 * where it reads none.
 *
 * The manifest declares every command; the frontend scan only confirms that each id it
 * registers is declared, and the commands it registers are the frontend's. An id the manifest
 * names under `contributes.handlers` is a shared command the frontend handles, and is no command
 * of the plugin.
 */
export const getPluginCommands = async (target, plugins = PLUGINS) => {
  const found = []
  for (const { name, manifest } of readManifests(plugins)) {
    if (!manifest.targets.includes(target)) continue
    const at = new URL(`${name}/`, plugins)
    const all = manifest.contributes?.commands ?? []
    const handled = (manifest.contributes?.handlers ?? []).map((one) => one.command)
    const registered = (manifest.frontend ? readRegisteredIds(at, name) : []).filter(
      (id) => !handled.includes(id),
    )
    for (const id of registered) {
      if (!all.some((one) => one.id === id)) {
        fail(`${name} registers ${id}, which its manifest does not declare`)
      }
    }
    found.push({
      id: manifest.id,
      settings: manifest.settings
        .filter((one) => one.palette)
        .map((one) => ({ text: `${manifest.name}: ${one.title}`, group: 'settings' })),
      declared: all
        .filter((one) => !registered.includes(one.id) && !isOverVersion(one.when))
        .map((one) => ({ text: one.title, group: getWhenGroup(one.when) })),
      commands: all
        .filter((one) => registered.includes(one.id))
        .map((one) => ({
          id: one.id,
          text: one.title,
          group: getWhenGroup(one.when),
          ...(one.shortcut ? { chord: getShortcutChord(one.shortcut, name) } : {}),
        })),
    })
  }
  return found.sort((a, b) => (a.id < b.id ? -1 : a.id > b.id ? 1 : 0))
}

/** The manifests under the plugins folder; one that cannot be read is a ReadError. */
const readManifests = (plugins) => {
  try {
    return readPluginManifests(fileURLToPath(plugins))
  } catch (error) {
    if (error instanceof ManifestError) fail(error.message)
    throw error
  }
}

/**
 * The ids one plugin's frontend registers a handler for. Every `registerCommand` call of the
 * frontend's sources is read; one this cannot read an id out of is refused.
 */
const readRegisteredIds = (at, name) =>
  listRegistrations(fileURLToPath(new URL('frontend/', at))).map(({ id }) => {
    if (id === undefined) fail(`${name} registers a command this cannot read`)
    return id
  })
