/**
 * Builds and verifies a mirror directory: the assets at the paths of the inventory, and `mirror.json`,
 * which records the size, sha256 and source of each file the mirror holds.
 */
import { createHash } from 'node:crypto'
import {
  existsSync,
  mkdirSync,
  readFileSync,
  readdirSync,
  renameSync,
  writeFileSync,
} from 'node:fs'
import { dirname, join, relative } from 'node:path'
import { getMemberSha256 } from './archive.mjs'

export const RECORD_FILE = 'mirror.json'

const getSha256 = (data) => createHash('sha256').update(data).digest('hex')

export const readRecords = (root) => {
  const file = join(root, RECORD_FILE)
  return existsSync(file) ? JSON.parse(readFileSync(file, 'utf8')).files : {}
}

const writeRecords = (root, files) => {
  const sorted = Object.fromEntries(Object.entries(files).sort(([a], [b]) => a.localeCompare(b)))
  writeFileSync(
    join(root, RECORD_FILE),
    `${JSON.stringify({ version: 1, files: sorted }, null, 2)}\n`,
  )
}

const readMember = (asset, data) => {
  try {
    return { sum: getMemberSha256(data, asset.urls[0].split('?')[0], asset.member) }
  } catch (error) {
    return { error: error.message }
  }
}

/** Lists what is wrong with bytes held for an asset: its checksum, its member checksum. */
export const listDataProblems = (asset, data) => {
  const problems = []
  const sum = getSha256(data)
  if (sum !== asset.sha256) problems.push(`sha256 is ${sum}, the manifest declares ${asset.sha256}`)
  if (asset.member === '') return problems
  const member = readMember(asset, data)
  if (member.error) return [...problems, `the archive cannot be read: ${member.error}`]
  if (member.sum === null) problems.push(`the archive holds no ${asset.member}`)
  else if (member.sum !== asset.memberSha256)
    problems.push(`member sha256 is ${member.sum}, the manifest declares ${asset.memberSha256}`)
  return problems
}

const listFileProblems = (root, asset, record) => {
  const file = join(root, asset.path)
  if (!existsSync(file)) return ['is not in the mirror']
  const data = readFileSync(file)
  const problems = listDataProblems(asset, data)
  if (!record) problems.push(`has no entry in ${RECORD_FILE}`)
  else if (record.size !== data.length)
    problems.push(`is ${data.length} bytes, ${RECORD_FILE} records ${record.size}`)
  return problems
}

const listFiles = (dir, base = dir) =>
  readdirSync(dir, { withFileTypes: true }).flatMap((entry) =>
    entry.isDirectory()
      ? listFiles(join(dir, entry.name), base)
      : [relative(base, join(dir, entry.name))],
  )

/**
 * Checks a mirror against the assets the manifests declare, offline.
 *
 * @returns {{ problems: string[], extras: string[] }} a line per failing asset, and the files no asset names
 */
export const verifyMirror = (root, assets) => {
  const records = existsSync(root) ? readRecords(root) : {}
  const problems = assets.flatMap((asset) =>
    listFileProblems(root, asset, records[asset.path]).map((problem) => `${asset.path} ${problem}`),
  )
  const known = new Set([...assets.map((asset) => asset.path), RECORD_FILE])
  const extras = existsSync(root)
    ? listFiles(root)
        .filter((file) => !known.has(file))
        .sort()
    : []
  return { problems, extras }
}

const download = async (url, fetchUrl) => {
  const response = await fetchUrl(url)
  if (!response.ok) throw new Error(`${url}: status ${response.status}`)
  return Buffer.from(await response.arrayBuffer())
}

const fetchAsset = async (asset, fetchUrl) => {
  const errors = []
  for (const url of asset.urls) {
    try {
      const data = await download(url, fetchUrl)
      const problems = listDataProblems(asset, data)
      if (problems.length === 0) return { data, url }
      errors.push(`${url}: ${problems.join('; ')}`)
    } catch (error) {
      errors.push(error.message)
    }
  }
  return { errors }
}

const storeFile = (root, asset, data) => {
  const file = join(root, asset.path)
  mkdirSync(dirname(file), { recursive: true })
  writeFileSync(`${file}.part`, data)
  renameSync(`${file}.part`, file)
}

/**
 * Downloads the assets missing from the mirror, in the order of their URLs, and keeps a file only
 * when its checksums match the manifest. A file the mirror already holds and verifies is not fetched.
 *
 * @param {string} root - the mirror directory
 * @param {object[]} assets - the inventory
 * @param {(url: string) => Promise<Response>} fetchUrl - how a URL is fetched
 * @returns {Promise<{ mirrored: string[], kept: string[], failed: string[] }>} the paths mirrored now, already held, and a line for each asset no URL could supply
 */
export const fillMirror = async (root, assets, fetchUrl) => {
  mkdirSync(root, { recursive: true })
  const records = readRecords(root)
  const result = { mirrored: [], kept: [], failed: [] }
  for (const asset of assets) {
    if (listFileProblems(root, asset, records[asset.path]).length === 0) {
      result.kept.push(asset.path)
      continue
    }
    const got = await fetchAsset(asset, fetchUrl)
    if (got.errors) {
      result.failed.push(`${asset.path}: ${got.errors.join(' | ')}`)
      continue
    }
    storeFile(root, asset, got.data)
    records[asset.path] = { size: got.data.length, sha256: asset.sha256, source: got.url }
    result.mirrored.push(asset.path)
  }
  writeRecords(root, records)
  return result
}

export const listMirrorUrls = (base, assets) =>
  assets.map((asset) => ({ path: asset.path, url: `${base.replace(/\/+$/, '')}/${asset.path}` }))
