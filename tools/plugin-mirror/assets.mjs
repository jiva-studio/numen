/**
 * The inventory of the assets plugin manifests download: every dependency source with its checksum,
 * platforms and consumers, and the place each one has in a mirror.
 */
import { existsSync, readFileSync, readdirSync } from 'node:fs'
import { basename, join } from 'node:path'
import { fileURLToPath } from 'node:url'

export const PLUGINS_DIR = fileURLToPath(new URL('../../plugins/', import.meta.url))
export const INVENTORY_FILE = fileURLToPath(new URL('./inventory.json', import.meta.url))

const ANY_PLATFORM = 'any'
const SHA256_PATTERN = /^[0-9a-f]{64}$/

export class AssetError extends Error {}

const isObject = (value) => value !== null && typeof value === 'object' && !Array.isArray(value)

const listUrls = (source) => {
  if (Array.isArray(source.sourceUrls) && source.sourceUrls.length > 0) return source.sourceUrls
  return typeof source.sourceUrl === 'string' && source.sourceUrl !== '' ? [source.sourceUrl] : []
}

const listDependencySources = (dependency) => {
  const sources = Array.isArray(dependency.sources) ? dependency.sources : [dependency]
  return sources
    .map((source) => ({
      platform: source.platform ?? ANY_PLATFORM,
      urls: listUrls(source),
      sha256: source.checksumSha256 ?? '',
      member: source.archiveMember ?? '',
      memberSha256: source.memberSha256 ?? '',
    }))
    .filter((source) => source.urls.length > 0)
}

const listDependencies = (value, found = []) => {
  if (Array.isArray(value)) {
    for (const one of value) listDependencies(one, found)
    return found
  }
  if (!isObject(value)) return found
  for (const [key, child] of Object.entries(value)) {
    if (key === 'dependencies' && Array.isArray(child)) {
      found.push(...child.filter((one) => isObject(one) && typeof one.id === 'string'))
    }
    listDependencies(child, found)
  }
  return found
}

export const getMirrorPath = (asset) => {
  const name = basename(new URL(asset.urls[0]).pathname)
  const platform = asset.platform.replace('/', '-')
  return [asset.id, asset.version || 'unversioned', platform, name].join('/')
}

const getKey = (asset) =>
  [asset.id, asset.version, asset.platform, asset.sha256, asset.member, asset.memberSha256].join(
    '|',
  )

const mergeAsset = (known, asset, consumer) => {
  for (const url of asset.urls) if (!known.urls.includes(url)) known.urls.push(url)
  if (!known.consumers.includes(consumer)) known.consumers.push(consumer)
}

const checkAsset = (asset, consumer) => {
  if (!SHA256_PATTERN.test(asset.sha256)) {
    throw new AssetError(`${consumer}: ${asset.id} ${asset.platform} has no sha256 checksum`)
  }
}

/**
 * Lists the unique assets of the plugins, ordered by path.
 *
 * @param {{ name: string, document: Record<string, unknown> }[]} plugins - the plugin folders and their manifests
 * @returns {object[]} each asset: id, kind, version, platform, urls, sha256, member, memberSha256, consumers, path
 */
export const listAssets = (plugins) => {
  const assets = new Map()
  for (const plugin of plugins) {
    for (const dependency of listDependencies(plugin.document)) {
      for (const source of listDependencySources(dependency)) {
        const asset = {
          id: dependency.id,
          kind: dependency.kind ?? '',
          version: dependency.version ?? '',
          ...source,
          consumers: [plugin.name],
        }
        checkAsset(asset, plugin.name)
        const known = assets.get(getKey(asset))
        if (known) mergeAsset(known, asset, plugin.name)
        else assets.set(getKey(asset), asset)
      }
    }
  }
  return placeAssets([...assets.values()])
}

const placeAssets = (assets) => {
  const taken = new Map()
  const placed = assets.map((asset) => ({
    ...asset,
    consumers: [...asset.consumers].sort(),
    path: getMirrorPath(asset),
  }))
  for (const asset of placed) {
    if (taken.has(asset.path)) {
      throw new AssetError(
        `${asset.path} is the mirror path of two assets with different checksums`,
      )
    }
    taken.set(asset.path, asset)
  }
  return placed.sort((a, b) => a.path.localeCompare(b.path))
}

export const readPluginDocuments = (pluginsDir = PLUGINS_DIR) =>
  readdirSync(pluginsDir, { withFileTypes: true })
    .filter(
      (entry) => entry.isDirectory() && existsSync(join(pluginsDir, entry.name, 'manifest.json')),
    )
    .map((entry) => ({
      name: entry.name,
      document: JSON.parse(readFileSync(join(pluginsDir, entry.name, 'manifest.json'), 'utf8')),
    }))
    .sort((a, b) => a.name.localeCompare(b.name))

export const formatInventory = (assets) => `${JSON.stringify({ version: 1, assets }, null, 2)}\n`

export const readInventory = (file = INVENTORY_FILE) =>
  existsSync(file) ? JSON.parse(readFileSync(file, 'utf8')).assets : []

/** Lists what differs between the committed inventory and the assets the manifests declare. */
export const listInventoryDrift = (declared, recorded) => {
  const byPath = new Map(recorded.map((asset) => [asset.path, asset]))
  const problems = []
  for (const asset of declared) {
    const known = byPath.get(asset.path)
    if (!known)
      problems.push(
        `${asset.path} is declared by ${asset.consumers.join(', ')} and not inventoried`,
      )
    else if (JSON.stringify(known) !== JSON.stringify(asset))
      problems.push(`${asset.path} differs from the manifests`)
    byPath.delete(asset.path)
  }
  for (const path of byPath.keys())
    problems.push(`${path} is inventoried and no manifest declares it`)
  return problems
}
