#!/usr/bin/env node
/**
 * The downloads storage zone (dl.numen.md) as the registry and the quality files use it: read an
 * object, write a file or a folder, merge a quality file, and purge served URLs.
 *
 *   node storage.mjs get <remote path> <local file>       absent object: no file, exit 0
 *   node storage.mjs put <local file or folder> <remote path>
 *   node storage.mjs merge-quality <mutation|coverage> <local file>
 *   node storage.mjs purge <remote path>...
 *
 * Environment: BUNNY_STORAGE_ENDPOINT, BUNNY_STORAGE_ZONE, BUNNY_STORAGE_KEY; `merge-quality` and
 * `purge` also BUNNY_API_KEY and BUNNY_SERVED_FROM, and purge nothing when either is empty.
 */
import { mkdirSync, readFileSync, readdirSync, statSync, writeFileSync } from 'node:fs'
import { dirname, join, relative, sep } from 'node:path'
import { fileURLToPath } from 'node:url'
import { QUALITY_FILES, formatQualityFile, mergeQualityFile } from '../shared/plugin-quality.mjs'

/** The zone named by the environment; an error names each missing variable. */
export const getStorage = (env = process.env) => {
  const names = ['BUNNY_STORAGE_ENDPOINT', 'BUNNY_STORAGE_ZONE', 'BUNNY_STORAGE_KEY']
  const missing = names.filter((name) => !env[name])
  if (missing.length > 0) throw new Error(`storage is not configured: ${missing.join(', ')}`)
  return {
    base: `https://${env.BUNNY_STORAGE_ENDPOINT}/${env.BUNNY_STORAGE_ZONE}`,
    key: env.BUNNY_STORAGE_KEY,
  }
}

/** The bytes of the object at `path`, or null when the zone has none. */
export const readObject = async (storage, path, fetcher = fetch) => {
  const response = await fetcher(`${storage.base}/${path}`, { headers: { AccessKey: storage.key } })
  if (response.status === 404) return null
  if (!response.ok) throw new Error(`reading ${path} was answered with ${response.status}`)
  return Buffer.from(await response.arrayBuffer())
}

/** Writes `body` to the object at `path`. */
export const writeObject = async (storage, path, body, fetcher = fetch) => {
  const response = await fetcher(`${storage.base}/${path}`, {
    method: 'PUT',
    headers: { AccessKey: storage.key, 'Content-Type': 'application/octet-stream' },
    body,
  })
  if (!response.ok) throw new Error(`writing ${path} was answered with ${response.status}`)
}

/** Every file under `dir`, as paths relative to it with forward slashes. */
export const listFiles = (dir) =>
  readdirSync(dir, { recursive: true })
    .map((one) => join(dir, String(one)))
    .filter((file) => statSync(file).isFile())
    .map((file) => relative(dir, file).split(sep).join('/'))
    .sort()

/** Purges the served copies of `paths`; nothing happens without an API key and a served base. */
export const purgePaths = async (paths, env = process.env, fetcher = fetch) => {
  const servedFrom = env.BUNNY_SERVED_FROM?.replace(/\/+$/, '')
  if (!env.BUNNY_API_KEY || !servedFrom) return []
  const urls = paths.map((path) => `${servedFrom}/${path}`)
  for (const url of urls) {
    const response = await fetcher(`https://api.bunny.net/purge?url=${encodeURIComponent(url)}`, {
      method: 'POST',
      headers: { AccessKey: env.BUNNY_API_KEY },
    })
    if (!response.ok) throw new Error(`purging ${url} was answered with ${response.status}`)
  }
  return urls
}

const getObject = async (remote, local) => {
  const body = await readObject(getStorage(), remote)
  if (body === null) {
    console.log(`${remote}: absent`)
    return
  }
  mkdirSync(dirname(local), { recursive: true })
  writeFileSync(local, body)
  console.log(`${remote}: ${body.length} bytes to ${local}`)
}

const putObjects = async (local, remote) => {
  const storage = getStorage()
  if (statSync(local).isFile()) {
    await writeObject(storage, remote, readFileSync(local))
    console.log(`put ${remote}`)
    return
  }
  for (const file of listFiles(local)) {
    await writeObject(storage, `${remote}/${file}`, readFileSync(join(local, file)))
    console.log(`put ${remote}/${file}`)
  }
}

const mergeQuality = async (kind, local) => {
  const path = QUALITY_FILES[kind]
  if (!path) throw new Error(`no quality file called ${kind}`)
  const storage = getStorage()
  const stored = await readObject(storage, path)
  const measured = JSON.parse(readFileSync(local, 'utf8'))
  const merged = mergeQualityFile(
    stored === null ? null : JSON.parse(stored.toString('utf8')),
    measured.targets,
    measured.measuredAt,
  )
  await writeObject(storage, path, formatQualityFile(merged))
  console.log(`${path}: ${measured.targets.length} measured, ${merged.targets.length} stored`)
  for (const url of await purgePaths([path])) console.log(`purged ${url}`)
}

const COMMANDS = {
  get: ([remote, local]) => getObject(remote, local),
  put: ([local, remote]) => putObjects(local, remote),
  'merge-quality': ([kind, local]) => mergeQuality(kind, local),
  purge: async (paths) => {
    for (const url of await purgePaths(paths)) console.log(`purged ${url}`)
  },
}

const main = async () => {
  const [command, ...args] = process.argv.slice(2)
  const run = COMMANDS[command]
  if (!run) throw new Error(`usage: storage.mjs ${Object.keys(COMMANDS).join('|')} ...`)
  await run(args)
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  main().catch((error) => {
    console.error(error.message)
    process.exit(1)
  })
}
