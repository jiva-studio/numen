/**
 * The screenshots the stories declare, read from the story sources with the TypeScript parser.
 *
 * A story is a screenshot when it carries the tag `screenshot` and an array of shots in
 * `parameters.screenshot`. A shot is written to `<into>/<name>-{light,dark}.webp`, where `into`
 * is the `screenshots/` folder of the plugin that holds the story, of the plugin `plugin` names, or
 * the `src/assets/` folder of the app `app` names. The values of a shot are literals.
 */
import { existsSync } from 'node:fs'
import { readFile } from 'node:fs/promises'
import { join, relative, sep } from 'node:path'
import ts from 'typescript'
import { hasPreset } from '../ci-stories/presets.mjs'
import { getStoryId, listStoryFiles, stories } from '../ci-stories/stories.mjs'
import { root } from '../shared/modules.mjs'

export const TAG = 'screenshot'

/** The folder each app writes its pictures into, relative to the repository. */
export const APPS = {
  docs: 'apps/docs/src/assets',
  landing: 'apps/landing/src/assets',
}

/** The window a shot is drawn in when it names no size. */
export const WINDOW = { width: 1180, height: 740 }

const KEYS = new Set(['name', 'plugin', 'app', 'width', 'height', 'over', 'preset', 'drawn'])
const NAME = /^[a-z0-9]+(-[a-z0-9]+)*$/

/** The folder of the screenshots of a plugin, relative to the repository. */
export const getPluginFolder = (plugin) => `plugins/${plugin}/screenshots`

class NotLiteral extends Error {}

const unwrap = (node) => {
  let at = node
  while (
    at &&
    (ts.isAsExpression(at) || ts.isSatisfiesExpression(at) || ts.isParenthesizedExpression(at))
  ) {
    at = at.expression
  }
  return at
}

const readLiteral = (node) => {
  const at = unwrap(node)
  if (ts.isStringLiteral(at) || ts.isNoSubstitutionTemplateLiteral(at)) return at.text
  if (ts.isNumericLiteral(at)) return Number(at.text)
  if (at.kind === ts.SyntaxKind.TrueKeyword) return true
  if (at.kind === ts.SyntaxKind.FalseKeyword) return false
  if (ts.isArrayLiteralExpression(at)) return at.elements.map(readLiteral)
  if (ts.isObjectLiteralExpression(at)) {
    const value = {}
    for (const property of at.properties) {
      if (!ts.isPropertyAssignment(property)) throw new NotLiteral()
      const key = property.name
      if (!ts.isIdentifier(key) && !ts.isStringLiteral(key)) throw new NotLiteral()
      value[key.text] = readLiteral(property.initializer)
    }
    return value
  }
  throw new NotLiteral()
}

const getProperty = (object, key) =>
  object.properties.find(
    (property) =>
      ts.isPropertyAssignment(property) &&
      (ts.isIdentifier(property.name) || ts.isStringLiteral(property.name)) &&
      property.name.text === key,
  )?.initializer

const getTags = (object) => {
  const tags = object && getProperty(object, 'tags')
  if (!tags) return []
  try {
    const value = readLiteral(tags)
    return Array.isArray(value) ? value : []
  } catch {
    return []
  }
}

/** The meta object, the exported story objects and their names, of one story source. */
const readStoryFile = (path, text) => {
  const source = ts.createSourceFile(path, text, ts.ScriptTarget.Latest, true, ts.ScriptKind.TS)
  const objects = new Map()
  const exported = []
  let defaultName
  for (const statement of source.statements) {
    if (ts.isExportAssignment(statement)) {
      const value = unwrap(statement.expression)
      if (ts.isIdentifier(value)) defaultName = value.text
      else if (ts.isObjectLiteralExpression(value)) objects.set('default', value)
      continue
    }
    if (!ts.isVariableStatement(statement)) continue
    const isExported = statement.modifiers?.some((one) => one.kind === ts.SyntaxKind.ExportKeyword)
    for (const declaration of statement.declarationList.declarations) {
      if (!ts.isIdentifier(declaration.name)) continue
      const name = declaration.name.text
      const value = declaration.initializer && unwrap(declaration.initializer)
      if (value && ts.isObjectLiteralExpression(value)) objects.set(name, value)
      if (isExported && /^[A-Z]/.test(name)) exported.push(name)
    }
  }
  const meta = objects.get(defaultName ?? 'default')
  const titleNode = meta && getProperty(meta, 'title')
  let title
  try {
    title = titleNode ? readLiteral(titleNode) : undefined
  } catch {
    title = undefined
  }
  return {
    title,
    meta,
    stories: exported.map((name) => ({ name, object: objects.get(name) })),
  }
}

/** The plugin folder that holds a story file, or undefined outside `plugins/`. */
const getHomePlugin = (file) => {
  const parts = file.split(sep)
  return parts[0] === 'plugins' ? parts[1] : undefined
}

const hasPlugin = (plugin) => existsSync(join(root, 'plugins', plugin, 'manifest.json'))

/** The faults of one declared shot, and the folder it is written into when it has none. */
const readShot = (story, home, shot) => {
  const faults = []
  const at = `${story}: shot ${JSON.stringify(shot)}`
  if (typeof shot !== 'object' || shot === null || Array.isArray(shot)) {
    return { faults: [`${at} is not an object`] }
  }
  for (const key of Object.keys(shot)) {
    if (!KEYS.has(key)) faults.push(`${at} has the unknown key ${key}`)
  }
  if (typeof shot.name !== 'string' || !NAME.test(shot.name)) {
    faults.push(`${at} needs a name of lower-case words joined by dashes`)
  }
  if (shot.plugin !== undefined && shot.app !== undefined) {
    faults.push(`${at} names both a plugin and an app`)
  }
  if (shot.app !== undefined && !Object.hasOwn(APPS, shot.app)) {
    faults.push(`${at} names the app ${shot.app}, which is none of ${Object.keys(APPS).join(', ')}`)
  }
  if (shot.plugin !== undefined && (typeof shot.plugin !== 'string' || !hasPlugin(shot.plugin))) {
    faults.push(`${at} names the plugin ${shot.plugin}, which has no folder under plugins/`)
  }
  if (shot.plugin === undefined && shot.app === undefined && home === undefined) {
    faults.push(`${at} stands outside plugins/ and names neither a plugin nor an app`)
  }
  for (const key of ['width', 'height', 'drawn']) {
    if (shot[key] !== undefined && !(Number.isInteger(shot[key]) && shot[key] > 0)) {
      faults.push(`${at} has a ${key} that is not a positive whole number`)
    }
  }
  if (shot.over !== undefined && typeof shot.over !== 'string') {
    faults.push(`${at} has an over that is not a selector`)
  }
  if (shot.preset !== undefined && !(typeof shot.preset === 'string' && hasPreset(shot.preset))) {
    faults.push(`${at} names the palette ${shot.preset}, which themes-presets does not hold`)
  }
  if (faults.length > 0) return { faults }
  const into = shot.app ? APPS[shot.app] : getPluginFolder(shot.plugin ?? home)
  return {
    faults,
    shot: {
      story,
      name: shot.name,
      into,
      width: shot.width ?? WINDOW.width,
      height: shot.height ?? WINDOW.height,
      ...(shot.over ? { over: shot.over } : {}),
      ...(shot.preset ? { preset: shot.preset } : {}),
      ...(shot.drawn ? { drawn: shot.drawn } : {}),
    },
  }
}

/**
 * The shots of one story source at `file` (relative to the repository), and what is wrong with
 * its declarations.
 */
export const readShotsOf = (file, text) => {
  const { title, meta, stories } = readStoryFile(file, text)
  const metaTagged = getTags(meta).includes(TAG)
  const home = getHomePlugin(file)
  const shots = []
  const faults = []
  for (const { name, object } of stories) {
    const isTagged = metaTagged || getTags(object).includes(TAG)
    const parameters = object && getProperty(object, 'parameters')
    const declared =
      parameters && ts.isObjectLiteralExpression(unwrap(parameters))
        ? getProperty(unwrap(parameters), TAG)
        : undefined
    if (!isTagged && !declared) continue
    const story = title ? getStoryId(title, name) : `${file}#${name}`
    if (!title) {
      faults.push(`${story} has no title in its meta, so its id cannot be read`)
      continue
    }
    if (!declared) {
      faults.push(`${story} is tagged ${TAG} and declares no parameters.${TAG}`)
      continue
    }
    if (!isTagged) {
      faults.push(`${story} declares parameters.${TAG} and is not tagged ${TAG}`)
      continue
    }
    let value
    try {
      value = readLiteral(declared)
    } catch {
      faults.push(`${story}: parameters.${TAG} is not written in literals`)
      continue
    }
    if (!Array.isArray(value) || value.length === 0) {
      faults.push(`${story}: parameters.${TAG} is not a list of shots`)
      continue
    }
    for (const one of value) {
      const read = readShot(story, home, one)
      faults.push(...read.faults)
      if (read.shot) shots.push({ ...read.shot, file })
    }
  }
  return { shots, faults }
}

/**
 * Every shot the tree declares, with the Storybook it is drawn in, and what is wrong with the
 * declarations: a fault of a shot, a story two Storybooks declare, two shots written to one file.
 */
export const readShots = async () => {
  const shots = []
  const faults = []
  for (const { path, book } of await listStoryFiles()) {
    const file = relative(root, path)
    const read = readShotsOf(file, await readFile(path, 'utf8'))
    faults.push(...read.faults)
    for (const shot of read.shots) shots.push({ ...shot, book })
  }
  const declared = await stories()
  for (const story of new Set(shots.map((shot) => shot.story))) {
    if (declared.get(story) === 'both') {
      faults.push(`${story} is declared in two Storybooks, so it cannot be told which to start`)
    }
  }
  const written = new Map()
  for (const shot of shots) {
    const at = `${shot.into}/${shot.name}`
    if (written.has(at)) faults.push(`${written.get(at)} and ${shot.story} both write ${at}`)
    else written.set(at, shot.story)
  }
  return { shots, faults }
}

/** The two files a shot writes, relative to the repository. */
export const getFiles = (shot) =>
  ['light', 'dark'].map((theme) => `${shot.into}/${shot.name}-${theme}.webp`)
