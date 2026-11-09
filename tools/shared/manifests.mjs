/**
 * The one place the tools read a plugin's manifest.json.
 *
 * A manifest is read against the schema generated from `libs/core/protocol/proto/numen/v1/manifest.proto`:
 * a value of the wrong type is an error, and a key the schema does not name is left out of the message
 * and listed as a warning. Every tool takes the parsed message, the JSON document and the warnings
 * from here and parses nothing itself.
 */
import { existsSync, readFileSync, readdirSync } from 'node:fs'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { fromJson } from '@bufbuild/protobuf'
import {
  ManifestSchema,
  ManifestSettingOptionSchema,
} from '../../libs/core/protocol/src/numen/v1/manifest_pb.ts'

/** The name of the file a plugin declares itself in. */
export const MANIFEST_FILE = 'manifest.json'

/** The plugins folder of the repository. */
export const PLUGINS_DIR = fileURLToPath(new URL('../../plugins/', import.meta.url))

/** A manifest that cannot be read: it is not JSON, it does not fit the schema, or it names no plugin. */
export class ManifestError extends Error {}

const LIST_VALUE = 'google.protobuf.ListValue'

const isObject = (value) => value !== null && typeof value === 'object' && !Array.isArray(value)

/** The schema message a value of the field is read as, or null for a scalar and a well-known type. */
const getFieldMessage = (field) => {
  const message =
    field.fieldKind === 'map' ? field.mapKind === 'message' && field.message : field.message
  if (!message) return null
  if (message.typeName === LIST_VALUE) return ManifestSettingOptionSchema
  return message.typeName.startsWith('numen.v1.') ? message : null
}

/** The unknown keys of one entry; an entry of a map of option lists is a list of options. */
const listEntryKeys = (entry, message, path) =>
  Array.isArray(entry)
    ? entry.flatMap((one, index) => listUnknownKeys(one, message, `${path}[${index}]`))
    : listUnknownKeys(entry, message, path)

/**
 * The paths of the keys of `value` that the message does not name by its JSON name, at every depth.
 * Paths are written as the Go reader writes them: `settings[0].mood`.
 *
 * @param {unknown} value - the parsed JSON
 * @param {import('@bufbuild/protobuf').DescMessage} message - the schema the value is read as
 * @param {string} path - where the value is, empty for the document
 * @returns {string[]} the unknown paths, ordered by key at each object
 */
export const listUnknownKeys = (value, message, path = '') => {
  if (!isObject(value)) return []
  const found = []
  for (const key of Object.keys(value).sort((a, b) => a.localeCompare(b))) {
    const here = path === '' ? key : `${path}.${key}`
    const field = message.fields.find((one) => one.jsonName === key)
    if (field === undefined) {
      found.push(here)
      continue
    }
    const child = getFieldMessage(field)
    if (child === null) continue
    const item = value[key]
    if (field.fieldKind === 'map') {
      if (isObject(item)) {
        for (const name of Object.keys(item).sort((a, b) => a.localeCompare(b))) {
          found.push(...listEntryKeys(item[name], child, `${here}.${name}`))
        }
      }
    } else if (field.fieldKind === 'list' || field.message.typeName === LIST_VALUE) {
      if (Array.isArray(item)) {
        item.forEach((one, index) =>
          found.push(...listUnknownKeys(one, child, `${here}[${index}]`)),
        )
      }
    } else {
      found.push(...listUnknownKeys(item, child, here))
    }
  }
  return found
}

/**
 * A manifest.json text read against the generated schema.
 *
 * @param {string} text - the content of the file
 * @param {string} source - where the text comes from, named in the error
 * @returns {{ manifest: import('@bufbuild/protobuf').MessageShape<typeof ManifestSchema>, document: Record<string, unknown>, warnings: string[] }}
 *   the schema's message, the JSON document it was read from, and a warning for each key the schema
 *   does not name
 */
export const parseManifest = (text, source = MANIFEST_FILE) => {
  let document
  try {
    document = JSON.parse(text)
  } catch (error) {
    throw new ManifestError(`${source} is not JSON: ${error.message}`, { cause: error })
  }
  if (document === null || typeof document !== 'object' || Array.isArray(document)) {
    throw new ManifestError(`${source} is not a JSON object`)
  }
  let manifest
  try {
    manifest = fromJson(ManifestSchema, document, { ignoreUnknownFields: true })
  } catch (error) {
    throw new ManifestError(`${source} does not fit the manifest schema: ${error.message}`, {
      cause: error,
    })
  }
  const missing = ['id', 'name', 'version'].filter((field) => manifest[field] === '')
  if (missing.length > 0) {
    throw new ManifestError(`${source} has no ${missing.join(', ')}`)
  }
  const warnings = listUnknownKeys(document, ManifestSchema).map((path) => `unknown key "${path}"`)
  return { manifest, document, warnings }
}

/**
 * The version a manifest.json text declares, read without the schema: a revision older than the
 * schema still has one, and the tools that compare versions across revisions need nothing else.
 *
 * @param {string} text - the content of the file
 * @param {string} source - where the text comes from, named in the error
 * @returns {string} the version
 */
export const readManifestVersion = (text, source = MANIFEST_FILE) => {
  let document
  try {
    document = JSON.parse(text)
  } catch (error) {
    throw new ManifestError(`${source} is not JSON: ${error.message}`, { cause: error })
  }
  if (typeof document?.version !== 'string' || document.version === '') {
    throw new ManifestError(`${source} has no version`)
  }
  return document.version
}

/** The manifest at `file`, read and checked. */
export const readManifestFile = (file) => parseManifest(readFileSync(file, 'utf8'), file)

/** The folders under `pluginsDir` that hold a manifest.json, by name. */
export const listPluginNames = (pluginsDir = PLUGINS_DIR) =>
  readdirSync(pluginsDir, { withFileTypes: true })
    .filter(
      (entry) => entry.isDirectory() && existsSync(join(pluginsDir, entry.name, MANIFEST_FILE)),
    )
    .map((entry) => entry.name)
    .sort()

/**
 * Every plugin under `pluginsDir`: its folder name, the path of its manifest, the schema's message
 * and the JSON document.
 */
export const readPluginManifests = (pluginsDir = PLUGINS_DIR) =>
  listPluginNames(pluginsDir).map((name) => {
    const file = join(pluginsDir, name, MANIFEST_FILE)
    return { name, file, ...readManifestFile(file) }
  })
