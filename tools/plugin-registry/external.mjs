/**
 * The registry entry of a plugin published from its own GitHub repository.
 *
 * The repository follows `docs/plugins/registry.md`: `manifest.json`, `README.md` and
 * `screenshots/` at the tag of its latest GitHub release, platform artifacts on that release, and
 * an optional `quality.json`. The latest release is asked for with the ETag of the previous
 * registry. An unchanged release keeps the previous entry and downloads nothing: the answer is 304,
 * or it names the same tag and publication time (the ETag also changes with download counts).
 */
import { createHash } from 'node:crypto'
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { parseManifest } from '../shared/manifests.mjs'
import { findTool, runTool } from '../shared/tools.mjs'
import {
  checkAddedPermissions,
  checkArtifacts,
  checkHasTests,
  checkLicense,
  checkManifest,
  checkMutationScore,
  checkNetwork,
  checkReadme,
  checkScreenshots,
  checkStories,
  checkVulnerabilities,
  createCheck,
} from './checks.mjs'
import { describeDownloads } from './downloads.mjs'
import {
  describeManifest,
  listAddedPermissions,
  pairScreenshots,
  splitReadme,
  toManifestJson,
} from './entry.mjs'
import { getExternalQuality } from './quality.mjs'
import { inspectSource } from './source.mjs'

/** The storage folder the registry and the screenshots of external plugins are published to. */
export const REGISTRY_PATH = 'plugins/registry'

const API = 'https://api.github.com'
const RAW = 'https://raw.githubusercontent.com'
const RELEASES_KEPT = 30

const PLATFORMS = [
  { platform: 'macos', pattern: /(^|[-_.])(darwin|macos)([-_.]|$)/i },
  { platform: 'windows', pattern: /(^|[-_.])windows([-_.]|$)/i },
  { platform: 'linux', pattern: /(^|[-_.])linux([-_.]|$)/i },
]

const ARCHES = [
  { arch: 'arm64', pattern: /(^|[-_.])(arm64|aarch64)([-_.]|$)/i },
  { arch: 'amd64', pattern: /(^|[-_.])(amd64|x86_64|x64)([-_.]|$)/i },
  { arch: 'universal', pattern: /(^|[-_.])universal([-_.]|$)/i },
]

/**
 * A client of the GitHub REST API and of raw repository files.
 *
 * @param {{ token?: string, fetcher?: typeof fetch }} options
 */
export const createGitHub = ({ token, fetcher = fetch } = {}) => {
  const headers = {
    Accept: 'application/vnd.github+json',
    'X-GitHub-Api-Version': '2022-11-28',
    'User-Agent': 'numen-plugin-registry',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
  }
  return {
    /** The JSON at an API path: `{ status: 304 }` when `etag` still matches, `{ status: 404 }` when absent. */
    getJson: async (path, etag) => {
      const response = await fetcher(`${API}${path}`, {
        headers: etag ? { ...headers, 'If-None-Match': etag } : headers,
      })
      if (response.status === 304 || response.status === 404) return { status: response.status }
      if (!response.ok) throw new Error(`GitHub answered ${path} with ${response.status}`)
      return { status: 200, etag: response.headers.get('etag'), body: await response.json() }
    },
    /** A file of a repository at a ref, or null when it is absent. */
    getFile: async (repo, ref, path) => {
      const response = await fetcher(`${RAW}/${repo}/${ref}/${path}`, {
        headers: { 'User-Agent': headers['User-Agent'] },
      })
      if (response.status === 404) return null
      if (!response.ok) throw new Error(`${repo}@${ref}:${path} was answered with ${response.status}`)
      return Buffer.from(await response.arrayBuffer())
    },
    /** The bytes at a download URL. */
    getBytes: async (url) => {
      const response = await fetcher(url, { headers: { 'User-Agent': headers['User-Agent'] } })
      if (!response.ok) throw new Error(`${url} was answered with ${response.status}`)
      return Buffer.from(await response.arrayBuffer())
    },
  }
}

/** The platform and architecture an artifact name states, or null when it states no platform. */
export const readArtifactPlatform = (name) => {
  const platform = PLATFORMS.find((one) => one.pattern.test(name))?.platform
  if (!platform) return null
  return { platform, arch: ARCHES.find((one) => one.pattern.test(name))?.arch ?? null }
}

const getSha256 = (bytes) => createHash('sha256').update(bytes).digest('hex')

/**
 * Verifies the build provenance of an artifact file with `gh attestation verify`; skipped where gh
 * is not installed.
 */
export const verifyAttestation = async ({ repo, file }) => {
  if (!findTool('gh')) return createCheck('attestation', 'skipped', 'gh is not installed.')
  const result = runTool('gh', ['attestation', 'verify', file, '--repo', repo], { encoding: 'utf8' })
  return result.status === 0
    ? createCheck('attestation', 'pass', 'Build provenance verified.')
    : createCheck('attestation', 'fail', (result.stderr || 'Not verified.').trim())
}

/**
 * Reads the source at the release tag, cloned with git, as an internal plugin's folder is read: its
 * dependencies, their known vulnerabilities, its network use and its stories. Null where git is
 * not installed or the tag cannot be cloned.
 */
export const inspectRelease = async ({ repo, tag, json, readers }) => {
  if (!findTool('git')) return null
  const dir = mkdtempSync(join(tmpdir(), 'plugin-source-'))
  try {
    const clone = runTool(
      'git',
      ['clone', '--quiet', '--depth', '1', '--branch', tag, `https://github.com/${repo}.git`, dir],
      { encoding: 'utf8' },
    )
    if (clone.status !== 0) return null
    return await inspectSource({ dir, lockRoot: dir, json, readers })
  } finally {
    rmSync(dir, { recursive: true, force: true })
  }
}

/** The default checks that need outside tools; a test passes its own. */
export const DEFAULT_VERIFIERS = {
  canVerifyAttestation: () => findTool('gh') !== null,
  verifyAttestation,
  inspectRelease,
}

const summarizeAttestations = (checks, canVerify) => {
  if (!canVerify) return createCheck('attestation', 'skipped', 'gh is not installed.')
  if (checks.length === 0) return createCheck('attestation', 'skipped', 'No artifacts.')
  const failed = checks.find((one) => one.status === 'fail')
  if (failed) return failed
  return checks.every((one) => one.status === 'pass')
    ? createCheck('attestation', 'pass', `${checks.length} artifacts with verified provenance.`)
    : checks[0]
}

const describeArtifacts = async (release, repo, github, verifiers) => {
  const artifacts = []
  const attestations = []
  const canVerify = verifiers.canVerifyAttestation()
  const work = mkdtempSync(join(tmpdir(), 'plugin-artifacts-'))
  try {
    for (const asset of release.assets ?? []) {
      const target = readArtifactPlatform(asset.name)
      if (!target) continue
      let sha256 = asset.digest?.startsWith('sha256:') ? asset.digest.slice('sha256:'.length) : null
      if (sha256 === null || canVerify) {
        const bytes = await github.getBytes(asset.browser_download_url)
        sha256 = getSha256(bytes)
        const file = join(work, asset.name)
        writeFileSync(file, bytes)
        if (canVerify) attestations.push(await verifiers.verifyAttestation({ repo, file }))
      }
      artifacts.push({
        name: asset.name,
        ...target,
        size: asset.size,
        sha256,
        url: asset.browser_download_url,
      })
    }
  } finally {
    rmSync(work, { recursive: true, force: true })
  }
  return { artifacts, attestation: summarizeAttestations(attestations, canVerify) }
}

const readScreenshots = async (source, tag, github, outDir) => {
  const listing = await github.getJson(
    `/repos/${source.repo}/contents/screenshots?ref=${encodeURIComponent(tag)}`,
  )
  if (listing.status !== 200 || !Array.isArray(listing.body)) return []
  const names = listing.body.filter((one) => one.type === 'file').map((one) => one.name)
  const getPath = (name) => `${REGISTRY_PATH}/${source.id}/screenshots/${name}`
  const screenshots = pairScreenshots(names, getPath)
  const kept = new Set(screenshots.flatMap((one) => [one.light, one.dark]).filter(Boolean))
  const folder = join(outDir, source.id, 'screenshots')
  for (const file of listing.body) {
    if (!kept.has(getPath(file.name))) continue
    mkdirSync(folder, { recursive: true })
    writeFileSync(join(folder, file.name), await github.getBytes(file.download_url))
  }
  return screenshots
}

const readChangelog = async (repo, github) => {
  const releases = await github.getJson(`/repos/${repo}/releases?per_page=${RELEASES_KEPT}`)
  if (releases.status !== 200) return []
  return releases.body
    .filter((one) => !one.draft)
    .map((one) => ({
      version: one.tag_name.replace(/^v/, ''),
      date: one.published_at,
      notes: one.body ?? '',
    }))
}

const readJsonFile = (bytes) => {
  if (bytes === null) return null
  try {
    return JSON.parse(bytes.toString('utf8'))
  } catch {
    return null
  }
}

/**
 * The entry of an external plugin, or the previous entry when its latest release is unchanged.
 * Screenshots are written under `outDir/<id>/screenshots/`, which is published at
 * `plugins/registry/`.
 *
 * @param {{ id: string, repo: string, author: string }} source - a row of `tools/plugin-registry/registry.json`
 * @param {object} context
 * @param {ReturnType<typeof createGitHub>} context.github
 * @param {Map<string, object>} context.previousById - the entries of the previous registry
 * @param {Record<string, string>} [context.icons] - lucide names by plugin id
 * @param {Map<string, number>} context.knownSizes - download sizes by SHA-256
 * @param {(url: string) => Promise<number | null>} context.getSize
 * @param {string} context.outDir - the folder the registry is written to
 * @param {typeof DEFAULT_VERIFIERS} [context.verifiers]
 * @param {ReturnType<typeof import('./source.mjs').createSourceReaders>} context.readers - the
 *   license and vulnerability readers
 * @returns {Promise<object>} the registry entry
 */
export const collectExternal = async (source, context) => {
  const { github, outDir, verifiers = DEFAULT_VERIFIERS } = context
  const previous = context.previousById.get(source.id)
  const latest = await github.getJson(`/repos/${source.repo}/releases/latest`, previous?.source.etag)
  if (latest.status === 304 && previous) return previous
  if (latest.status !== 200) throw new Error(`${source.repo} has no published release`)
  const release = latest.body
  const tag = release.tag_name
  if (previous?.source.tag === tag && previous.updatedAt === release.published_at) {
    return { ...previous, source: { ...previous.source, etag: latest.etag } }
  }

  const manifestBytes = await github.getFile(source.repo, tag, 'manifest.json')
  if (manifestBytes === null) throw new Error(`${source.repo}@${tag} has no manifest.json`)
  const { manifest, warnings } = parseManifest(
    manifestBytes.toString('utf8'),
    `${source.repo}@${tag}/manifest.json`,
  )
  const json = toManifestJson(manifest)
  if (json.id !== source.id) {
    throw new Error(`${source.repo}@${tag} declares the id ${json.id}, the registry lists ${source.id}`)
  }
  const fields = describeManifest(json, context.icons)
  const readmeBytes = await github.getFile(source.repo, tag, 'README.md')
  const readme = splitReadme(readmeBytes === null ? null : readmeBytes.toString('utf8'))
  const repository = await github.getJson(`/repos/${source.repo}`)
  const spdx = repository.body?.license?.spdx_id
  const license = spdx && spdx !== 'NOASSERTION' ? spdx : null
  const screenshots = await readScreenshots(source, tag, github, outDir)
  const { artifacts, attestation } = await describeArtifacts(release, source.repo, github, verifiers)
  const permissionsAdded = listAddedPermissions(fields, previous)
  const updatedAt = release.published_at ?? null
  const quality = getExternalQuality(
    readJsonFile(await github.getFile(source.repo, tag, 'quality.json')),
  )
  const inspected = await verifiers.inspectRelease({
    repo: source.repo,
    tag,
    json,
    readers: context.readers,
  })
  const scan = inspected?.scan ?? { vulnerabilities: null, reason: 'The source could not be cloned.' }

  return {
    ...fields,
    source: {
      kind: 'external',
      repo: source.repo,
      url: `https://github.com/${source.repo}`,
      tag,
      etag: latest.etag,
    },
    author: source.author,
    contributors: [],
    license,
    platforms: [...new Set(artifacts.map((one) => one.platform))].sort(),
    readme: readme && { intro: readme.intro, text: readme.text },
    screenshots,
    changelog: await readChangelog(source.repo, github),
    updatedAt,
    downloads: await describeDownloads(json.dependencies, context.knownSizes, context.getSize),
    permissionsAdded,
    artifacts,
    dependencies: inspected?.dependencies ?? { packages: [], indirect: { go: 0, npm: 0 } },
    vulnerabilities: scan.vulnerabilities ?? [],
    checks: [
      checkManifest(warnings),
      checkLicense(license),
      checkReadme(readme),
      checkScreenshots(screenshots),
      checkAddedPermissions(permissionsAdded),
      checkArtifacts(artifacts),
      attestation,
      checkHasTests(quality?.tests),
      checkMutationScore(quality?.entries),
      inspected && checkStories(inspected.stories),
      inspected && checkNetwork(inspected.network),
      checkVulnerabilities(scan),
    ].filter(Boolean),
    quality,
  }
}
