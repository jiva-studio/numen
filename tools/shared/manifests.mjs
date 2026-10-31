/**
 * The one place the tools read a plugin's manifest.json.
 *
 * A manifest is read against the schema generated from `libs/protocol/proto/numen/v1/manifest.proto`:
 * a value of the wrong type is an error, and keys the schema does not name are ignored. Every tool
 * takes the parsed message and the JSON document from here and parses nothing itself.
 */
import { existsSync, readFileSync, readdirSync } from 'node:fs'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { fromJson } from '@bufbuild/protobuf'
import { ManifestSchema } from '../../libs/protocol/src/numen/v1/manifest_pb.ts'

/** The name of the file a plugin declares itself in. */
export const MANIFEST_FILE = 'manifest.json'

/** The plugins folder of the repository. */
export const PLUGINS_DIR = fileURLToPath(new URL('../../plugins/', import.meta.url))

/** A manifest that cannot be read: it is not JSON, it does not fit the schema, or it names no plugin. */
export class ManifestError extends Error {}

/**
 * A manifest.json text read against the generated schema.
 *
 * @param {string} text - the content of the file
 * @param {string} source - where the text comes from, named in the error
 * @returns {{ manifest: import('@bufbuild/protobuf').MessageShape<typeof ManifestSchema>, document: Record<string, unknown> }}
 *   the schema's message and the JSON document it was read from
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
  return { manifest, document }
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
