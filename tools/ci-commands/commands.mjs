/**
 * The commands every plugin adds to a window's palette, read out of the
 * plugins' manifests and frontends.
 *
 * The manual lists them, and an application reaches no plugin, so the reading
 * stands here and the manual takes what it returns.
 */
import { readFile } from 'node:fs/promises'
import { fileURLToPath } from 'node:url'
import ts from 'typescript'
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
export const readBracketedText = (source, at) => {
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

const unwrapExpression = (node) =>
  node &&
  (ts.isAsExpression(node) || ts.isSatisfiesExpression(node) || ts.isParenthesizedExpression(node))
    ? unwrapExpression(node.expression)
    : node

/** The `WORDS` object literal declared in `sourceFile`, or the first object literal when none is named `WORDS`. */
const findWordsObject = (sourceFile) => {
  for (const statement of sourceFile.statements) {
    if (!ts.isVariableStatement(statement)) continue
    for (const declaration of statement.declarationList.declarations) {
      if (
        ts.isIdentifier(declaration.name) &&
        (declaration.name.text === 'WORDS' || declaration.name.text === 'PALETTE_WORDS')
      ) {
        const initializer = unwrapExpression(declaration.initializer)
        if (initializer && ts.isObjectLiteralExpression(initializer)) return initializer
      }
    }
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
  const sourceFile = ts.createSourceFile('words.ts', whole, ts.ScriptTarget.Latest, true)
  const words = findWordsObject(sourceFile)
  const property = words?.properties.find(
    (one) =>
      ts.isPropertyAssignment(one) &&
      (ts.isIdentifier(one.name) || ts.isStringLiteral(one.name)) &&
      one.name.text === name,
  )
  const value = unwrapExpression(property?.initializer)
  if (!value) fail(`the words say nothing under '${name}'`)

  if (ts.isStringLiteral(value) || ts.isNoSubstitutionTemplateLiteral(value)) {
    return value.text
  }

  if (
    ts.isArrowFunction(value) &&
    value.parameters.length > 0 &&
    ts.isIdentifier(value.parameters[0].name)
  ) {
    const parameter = value.parameters[0].name.text
    const body = unwrapExpression(value.body)
    if (body && (ts.isStringLiteral(body) || ts.isNoSubstitutionTemplateLiteral(body))) {
      return body.text
    }
    if (body && ts.isTemplateExpression(body)) {
      return (
        body.head.text +
        body.templateSpans
          .map((span) => {
            const expression = unwrapExpression(span.expression)
            const piece =
              ts.isIdentifier(expression) && expression.text === parameter
                ? `*${parameter}*`
                : `\${${span.expression.getText(sourceFile)}}`
            return piece + span.literal.text
          })
          .join('')
      )
    }
  }

  if (
    ts.isPropertyAccessExpression(value) &&
    ts.isIdentifier(value.expression) &&
    ts.isIdentifier(value.name)
  ) {
    const module = value.expression.text
    const key = value.name.text
    if (seen.has(module)) fail(`the words under '${name}' point at themselves`)

    const path = getWordsFile(sourceFile, module, at)
    return readWords(await readFile(path, 'utf8'), key, path, new Set([...seen, module]))
  }

  return fail(`the words say nothing under '${name}'`)
}

/** The file a module takes one set of words in from, as `WORDS` or `WORDS as <module>`. */
const getWordsFile = (sourceFile, module, at) => {
  for (const statement of sourceFile.statements) {
    if (!ts.isImportDeclaration(statement) || !ts.isStringLiteral(statement.moduleSpecifier))
      continue
    const bindings = statement.importClause?.namedBindings
    if (!bindings || !ts.isNamedImports(bindings)) continue
    const matches = bindings.elements.some(
      (specifier) =>
        (specifier.propertyName ?? specifier.name).text === 'WORDS' &&
        specifier.name.text === module,
    )
    if (matches) return new URL(`${statement.moduleSpecifier.text}.ts`, at)
  }
  return fail(`the words take nothing in as '${module}'`)
}

/**
 * The chord a plugin's keybinding is drawn as, such as `mod+shift+x`. The window
 * draws a key that is its modifier with a letter, and Shift or not; any
 * other key is refused, as the window would draw it as nothing.
 */
const getKeyChord = (key, where, known = ['mod', 'ctrl', 'control', 'cmd', 'shift']) => {
  const parts = key.toLowerCase().split('+')
  const letter = parts.at(-1)
  const modifiers = parts.slice(0, -1)
  if (!/^[a-z]$/.test(letter) || modifiers.some((one) => !known.includes(one))) {
    fail(`${where} binds the key '${key}', which the window draws as nothing`)
  }
  const alt = modifiers.includes('alt') ? { alt: true } : {}
  return { letter, shift: modifiers.includes('shift'), ...alt }
}

/** Whether a slot entry passes exactly the values a keybinding passes. */
const hasSameValues = (a = {}, b = {}) => {
  const names = Object.keys(a)
  return names.length === Object.keys(b).length && names.every((name) => a[name] === b[name])
}

/**
 * The keys a manifest binds to a command of the application with values, such as showing a note in
 * the plex, each named by the title of the slot entry that passes the same values and drawn with
 * Alt allowed.
 */
const getBoundKeys = (manifest, name) =>
  (manifest.contributes?.keybindings ?? [])
    .filter((one) => one.command.startsWith('app:') && Object.keys(one.values ?? {}).length > 0)
    .map((one) => {
      const entry = (manifest.contributes?.slots ?? []).find(
        (slot) =>
          slot.placed.case === 'command' &&
          slot.placed.value === one.command &&
          hasSameValues(slot.values, one.values),
      )
      if (!entry?.title) fail(`${name} binds '${one.key}' to a command no entry of a slot titles`)
      return {
        text: entry.title,
        group: /\btarget\.isNote\b/.test(one.when) ? 'note' : 'window',
        chord: getKeyChord(one.key, name, ['mod', 'ctrl', 'control', 'cmd', 'shift', 'alt']),
      }
    })

/** The key the manifest binds to its command of the id, or an empty string for none. */
const getKey = (manifest, id) =>
  (manifest.contributes?.keybindings ?? []).find((one) => one.command === id)?.key ?? ''

/** The palette group of a declared command by its when condition, as the window files it. */
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
 * - `declared`, each command of a plugin with a backend that its manifest declares and its
 *   frontend does not register, which the palette shows, leaving out the ones whose condition names
 *   a text version;
 * - `commands`, each command its manifest declares that its frontend registers a
 *   handler for, with the chord its keybinding is drawn as, over the note where its condition
 *   reads target.isNote and over the window where it does not;
 * - `bound`, each key the manifest binds to a command of the application with values, with the
 *   title the slot entry gives it and its chord;
 * - `handled`, the id of each shared command its manifest handles under
 *   `contributes.handlers`.
 *
 * A declared command is in the group the window files it under by its condition: over the note
 * where it reads target.isNote, over a file where it reads another field of target, and over the
 * window where it reads none.
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
        .filter(
          (one) => manifest.backend && !registered.includes(one.id) && !isOverVersion(one.when),
        )
        .map((one) => ({ text: one.title, group: getWhenGroup(one.when) })),
      commands: all
        .filter((one) => registered.includes(one.id))
        .map((one) => {
          const key = getKey(manifest, one.id)
          return {
            id: one.id,
            text: one.title,
            group: /\btarget\.isNote\b/.test(one.when) ? 'note' : 'window',
            ...(key ? { chord: getKeyChord(key, name) } : {}),
          }
        }),
      bound: getBoundKeys(manifest, name),
      handled,
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
 * The ids one plugin's frontend registers a handler for. Every `register.command` call of the
 * frontend's sources is read; one this cannot read an id out of is refused.
 */
const readRegisteredIds = (at, name) =>
  listRegistrations(fileURLToPath(new URL('frontend/', at))).map(({ id }) => {
    if (id === undefined) fail(`${name} registers a command this cannot read`)
    return id
  })
