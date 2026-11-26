/**
 * The changelog of a plugin of this repository, from the history of its folder: the commits that
 * touched it, grouped by the commit that changed the `version` of its manifest.
 */
import { readManifestVersion } from '../shared/manifests.mjs'
import { execTool } from '../shared/tools.mjs'

/** How many versions a changelog keeps, newest first. */
export const CHANGELOG_VERSIONS = 30

const runGit = (root, args) =>
  execTool('git', args, { cwd: root, encoding: 'utf8', maxBuffer: 64 * 1024 * 1024 })

/** The commits that touched `path`, newest first: hash, committer date and subject. */
export const readCommits = (root, path) =>
  runGit(root, ['log', '--format=%H%x09%cI%x09%s', '--', path])
    .split('\n')
    .filter(Boolean)
    .map((line) => {
      const [hash, date, ...subject] = line.split('\t')
      return { hash, date, subject: subject.join('\t') }
    })

const readVersions = (root, path) => {
  const hashes = runGit(root, ['log', '--format=%H', '-G"version"', '--', path])
    .split('\n')
    .filter(Boolean)
  const versions = new Map()
  for (const hash of hashes) {
    try {
      versions.set(hash, readManifestVersion(runGit(root, ['show', `${hash}:${path}`]), path))
    } catch {
      // A revision whose manifest has no version, or no manifest, starts no version.
    }
  }
  return versions
}

/**
 * Groups commits into versions. A commit whose manifest declares a version other than the one
 * before closes a version with itself and the commits since the previous one; the commits after the
 * last version change belong to no released version and are left out.
 *
 * @param {{ hash: string, date: string, subject: string }[]} commits - newest first
 * @param {Map<string, string>} versions - the version declared at a commit, where one may have changed
 * @param {number} limit - how many versions to keep
 * @returns {{ version: string, date: string, changes: string[] }[]} newest first
 */
export const groupByVersion = (commits, versions, limit = CHANGELOG_VERSIONS) => {
  const groups = []
  let current = null
  let pending = []
  for (const commit of [...commits].reverse()) {
    pending.push(commit.subject)
    const version = versions.get(commit.hash)
    if (version === undefined || version === current) continue
    groups.push({ version, date: commit.date, changes: pending.reverse() })
    current = version
    pending = []
  }
  return groups.reverse().slice(0, limit)
}

/**
 * The changelog and last change date of the plugin folder `folder` under `plugins/`.
 *
 * @param {string} root - the repository root
 * @param {string} folder - the plugin folder name
 * @returns {{ changelog: { version: string, date: string, changes: string[] }[], updatedAt: string | null }}
 */
export const readInternalHistory = (root, folder) => {
  const commits = readCommits(root, `plugins/${folder}`)
  const versions = readVersions(root, `plugins/${folder}/manifest.json`)
  return { changelog: groupByVersion(commits, versions), updatedAt: commits[0]?.date ?? null }
}
