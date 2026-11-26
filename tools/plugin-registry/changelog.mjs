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
 * last version change belong to no released version and are left out. Only features and fixes are
 * listed, without their type, and a version with neither is left out.
 *
 * @param {{ hash: string, date: string, subject: string }[]} commits - newest first
 * @param {Map<string, string>} versions - the version declared at a commit, where one may have changed
 * @param {number} limit - how many versions to keep
 * @returns {{ version: string, date: string, changes: string[] }[]} newest first
 */
/** The type of a commit a person using the plugin notices: a feature or a fix. */
const NOTICED = /^(feat|fix)(\([^)]*\))?!?: /

/** The subject of a commit a person notices without its type, or null for any other commit. */
export const readNoticedChange = (subject) => {
  const match = NOTICED.exec(subject)
  return match ? subject.slice(match[0].length) : null
}

export const groupByVersion = (commits, versions, limit = CHANGELOG_VERSIONS) => {
  const groups = []
  let current = null
  let pending = []
  for (const commit of [...commits].reverse()) {
    const change = readNoticedChange(commit.subject)
    if (change !== null) pending.push(change)
    const version = versions.get(commit.hash)
    if (version === undefined || version === current) continue
    if (pending.length > 0) groups.push({ version, date: commit.date, changes: pending.reverse() })
    current = version
    pending = []
  }
  return groups.reverse().slice(0, limit)
}

/**
 * The authors of `names`, one name per commit, with their commit counts: most commits first, then
 * by name.
 *
 * @param {string[]} names
 * @returns {{ name: string, commits: number }[]}
 */
export const countContributors = (names) => {
  const counts = new Map()
  for (const name of names) counts.set(name, (counts.get(name) ?? 0) + 1)
  return [...counts]
    .map(([name, commits]) => ({ name, commits }))
    .sort((a, b) => b.commits - a.commits || a.name.localeCompare(b.name))
}

/** The author names of the commits that touched `path`, with their commit counts. */
export const readContributors = (root, path) =>
  countContributors(runGit(root, ['log', '--format=%aN', '--', path]).split('\n').filter(Boolean))

/**
 * The changelog, last change date and contributors of the plugin folder `folder` under `plugins/`.
 *
 * @param {string} root - the repository root
 * @param {string} folder - the plugin folder name
 * @returns {{ changelog: { version: string, date: string, changes: string[] }[], updatedAt: string | null, contributors: { name: string, commits: number }[] }}
 */
export const readInternalHistory = (root, folder) => {
  const commits = readCommits(root, `plugins/${folder}`)
  const versions = readVersions(root, `plugins/${folder}/manifest.json`)
  return {
    changelog: groupByVersion(commits, versions),
    updatedAt: commits[0]?.date ?? null,
    contributors: readContributors(root, `plugins/${folder}`),
  }
}
