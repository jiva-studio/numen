/**
 * The registry entry of a plugin of this repository: a folder of `plugins/` with a manifest.json.
 */
import { existsSync, readFileSync, readdirSync } from 'node:fs'
import { join } from 'node:path'
import { MANIFEST_FILE, readManifestFile } from '../shared/manifests.mjs'
import { readInternalHistory } from './changelog.mjs'
import {
  checkLicense,
  checkManifest,
  checkPermissions,
  checkReadme,
  checkScreenshots,
  checkUpdated,
} from './checks.mjs'
import { describeDownloads } from './downloads.mjs'
import {
  describeManifest,
  listAddedPermissions,
  pairScreenshots,
  splitReadme,
  toManifestJson,
} from './entry.mjs'
import { getInternalQuality } from './quality.mjs'
import { countTests } from './tests.mjs'

/** The repository the internal plugins are published from. */
export const INTERNAL_REPO = 'jiva-studio/numen'

/** The author of the internal plugins. */
export const INTERNAL_AUTHOR = 'Jiva Studio'

/** The platforms the application ships on; an internal plugin ships with it. */
export const APP_PLATFORMS = ['macos', 'windows', 'linux']

/** The SPDX id of the repository's LICENSE, or null when it is not the AGPL version 3. */
export const readRepositoryLicense = (root) => {
  const file = join(root, 'LICENSE')
  if (!existsSync(file)) return null
  const text = readFileSync(file, 'utf8')
  return text.includes('GNU AFFERO GENERAL PUBLIC LICENSE') && text.includes('Version 3')
    ? 'AGPL-3.0'
    : null
}

const readOptional = (file) => (existsSync(file) ? readFileSync(file, 'utf8') : null)

const listScreenshotFiles = (dir) => (existsSync(dir) ? readdirSync(dir).sort() : [])

const COMMITS_PAGE = 100

/**
 * The GitHub account of each commit of `repo` that touched `path`, by commit hash, from the commits
 * API; a commit whose email belongs to no account is absent.
 *
 * @param {{ getJson: (path: string) => Promise<{ status: number, body?: unknown }> }} github
 * @returns {Promise<Map<string, { login: string, avatarUrl: string | null }>>}
 */
export const listCommitAccounts = async (github, repo, path) => {
  const accounts = new Map()
  for (let page = 1; ; page += 1) {
    const query = `path=${encodeURIComponent(path)}&per_page=${COMMITS_PAGE}&page=${page}`
    const { status, body } = await github.getJson(`/repos/${repo}/commits?${query}`)
    if (status !== 200 || !Array.isArray(body)) return accounts
    for (const commit of body) {
      const login = commit?.author?.login
      if (typeof commit?.sha === 'string' && typeof login === 'string') {
        accounts.set(commit.sha, { login, avatarUrl: commit.author.avatar_url ?? null })
      }
    }
    if (body.length < COMMITS_PAGE) return accounts
  }
}

/**
 * The entry of the plugin folder `folder`.
 *
 * @param {string} folder - the folder name under `plugins/`
 * @param {object} context
 * @param {string} context.root - the repository root
 * @param {string | null} context.license - the SPDX id of the repository
 * @param {Map<string, object>} context.previousById - the entries of the previous registry
 * @param {Map<string, number>} context.knownSizes - download sizes by SHA-256
 * @param {(url: string) => Promise<number | null>} context.getSize - the size of a file at a URL
 * @param {(path: string) => Promise<Map<string, { login: string, avatarUrl: string | null }>>} context.getCommitAccounts
 *   - the GitHub accounts of the commits that touched a path, by commit hash
 * @param {{ mutation: Map<string, object>, coverage: Map<string, object> }} context.quality
 * @returns {Promise<object>} the registry entry
 */
export const collectInternal = async (folder, context) => {
  const dir = join(context.root, 'plugins', folder)
  const { manifest, warnings } = readManifestFile(join(dir, MANIFEST_FILE))
  const json = toManifestJson(manifest)
  const fields = describeManifest(json)
  const readme = splitReadme(readOptional(join(dir, 'README.md')))
  const screenshots = pairScreenshots(
    listScreenshotFiles(join(dir, 'screenshots')),
    (name) => `plugins/${folder}/screenshots/${name}`,
  )
  const accounts = await context.getCommitAccounts(`plugins/${folder}`)
  const { changelog, updatedAt, contributors } = readInternalHistory(context.root, folder, accounts)
  const previous = context.previousById.get(fields.id)
  const permissionsAdded = listAddedPermissions(fields, previous)
  const downloads = await describeDownloads(json.dependencies, context.knownSizes, context.getSize)
  return {
    ...fields,
    source: {
      kind: 'internal',
      repo: INTERNAL_REPO,
      path: `plugins/${folder}`,
      url: `https://github.com/${INTERNAL_REPO}/tree/main/plugins/${folder}`,
    },
    author: INTERNAL_AUTHOR,
    contributors,
    license: context.license,
    platforms: APP_PLATFORMS,
    readme: readme && { intro: readme.intro, text: readme.text },
    screenshots,
    changelog,
    updatedAt,
    downloads,
    permissionsAdded,
    artifacts: [],
    checks: [
      checkManifest(warnings),
      checkLicense(context.license),
      checkReadme(readme),
      checkScreenshots(screenshots),
      checkPermissions(fields.permissions, permissionsAdded),
      checkUpdated(updatedAt),
    ],
    quality: getInternalQuality(folder, context.quality, countTests(dir)),
  }
}

