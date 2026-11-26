/**
 * The registry document: its sources, its content hash and the check that internal and external
 * plugins do not collide.
 */
import { createHash } from 'node:crypto'
import { existsSync, readFileSync } from 'node:fs'
import { OFFICIAL_PREFIX } from './entry.mjs'

/** The version of the registry document's shape. */
export const REGISTRY_SCHEMA = 1

/** The GitHub owner whose repositories may publish plugins under the official id prefix. */
export const OFFICIAL_OWNER = 'jiva-studio'

/** Keys whose values change without the catalog changing; the content hash leaves them out. */
const VOLATILE_KEYS = new Set(['generatedAt', 'contentHash', 'measuredAt', 'etag'])

const REPO = /^[A-Za-z0-9-]+\/[A-Za-z0-9._-]+$/

/**
 * The external sources listed in `file`: `{ "external": [{ "id", "repo", "author" }] }`. A missing
 * file lists none.
 *
 * @param {string} file
 * @returns {{ id: string, repo: string, author: string }[]}
 */
export const readSources = (file) => {
  if (!existsSync(file)) return []
  const document = JSON.parse(readFileSync(file, 'utf8'))
  const rows = document?.external
  if (!Array.isArray(rows)) throw new Error(`${file} has no "external" list`)
  for (const row of rows) {
    if (typeof row?.id !== 'string' || row.id === '') throw new Error(`${file}: a row has no id`)
    if (typeof row.repo !== 'string' || !REPO.test(row.repo)) {
      throw new Error(`${file}: ${row.id} has no repository of the form owner/name`)
    }
    if (typeof row.author !== 'string' || row.author === '') {
      throw new Error(`${file}: ${row.id} has no author`)
    }
    if (row.id.startsWith(OFFICIAL_PREFIX) && row.repo.split('/')[0] !== OFFICIAL_OWNER) {
      throw new Error(`${file}: ${row.id} uses ${OFFICIAL_PREFIX} from outside ${OFFICIAL_OWNER}`)
    }
  }
  return rows.map(({ id, repo, author }) => ({ id, repo, author }))
}

/**
 * The ids the catalog features, in order, listed in `file`: `{ "featured": ["<id>", …] }`. A
 * missing file or list features none.
 *
 * @param {string} file
 * @returns {string[]}
 */
export const readFeatured = (file) => {
  if (!existsSync(file)) return []
  const rows = JSON.parse(readFileSync(file, 'utf8'))?.featured ?? []
  if (!Array.isArray(rows) || !rows.every((id) => typeof id === 'string' && id !== '')) {
    throw new Error(`${file}: "featured" is not a list of plugin ids`)
  }
  return rows
}

/**
 * Throws when a featured id is not the id of a plugin of the registry.
 *
 * @param {string[]} featured
 * @param {{ id: string }[]} plugins
 */
export const assertFeaturedKnown = (featured, plugins) => {
  const ids = new Set(plugins.map((one) => one.id))
  const unknown = featured.filter((id) => !ids.has(id))
  if (unknown.length > 0) throw new Error(`featured ${unknown.join(', ')} is not in the registry`)
}

/**
 * Throws when an external source repeats an id, or takes the id or the folder name of an internal
 * plugin.
 *
 * @param {{ id: string, repo: string }[]} sources
 * @param {{ id: string, folder: string }[]} internals
 */
export const assertNoCollision = (sources, internals) => {
  const ids = new Set(internals.map((one) => one.id))
  const folders = new Set(internals.map((one) => one.folder))
  const seen = new Set()
  for (const source of sources) {
    const name = source.repo.split('/')[1]
    if (ids.has(source.id)) throw new Error(`external ${source.id} has the id of an internal plugin`)
    if (folders.has(source.id) || folders.has(name)) {
      throw new Error(`external ${source.id} (${source.repo}) has the name of an internal plugin folder`)
    }
    if (seen.has(source.id)) throw new Error(`external ${source.id} is listed twice`)
    seen.add(source.id)
  }
}

const stripVolatile = (value) => {
  if (Array.isArray(value)) return value.map(stripVolatile)
  if (value === null || typeof value !== 'object') return value
  return Object.fromEntries(
    Object.keys(value)
      .filter((key) => !VOLATILE_KEYS.has(key))
      .sort()
      .map((key) => [key, stripVolatile(value[key])]),
  )
}

/** The SHA-256 of a registry's content, without the generation time, measurement times and ETags. */
export const getContentHash = (registry) =>
  createHash('sha256').update(JSON.stringify(stripVolatile(registry))).digest('hex')

/**
 * The registry document of `plugins`, ordered by id, and the ids the catalog features.
 *
 * @param {object[]} plugins - the entries
 * @param {string} generatedAt - ISO time
 * @param {string[]} [featured] - plugin ids, in the order the catalog shows them
 */
export const createRegistry = (plugins, generatedAt, featured = []) => {
  const body = {
    schema: REGISTRY_SCHEMA,
    featured,
    plugins: [...plugins].sort((a, b) => a.id.localeCompare(b.id)),
  }
  return {
    schema: REGISTRY_SCHEMA,
    generatedAt,
    contentHash: getContentHash(body),
    featured,
    plugins: body.plugins,
  }
}

/** The text a registry file holds. */
export const formatRegistry = (registry) => `${JSON.stringify(registry, null, 2)}\n`
