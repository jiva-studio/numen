/**
 * The quality of a plugin as the registry shows it. A plugin of this repository takes the numbers
 * of the nightly mutation and coverage workflows (`tools/shared/plugin-quality.mjs`) and the tests
 * counted in its folder; a plugin of another repository takes the `quality.json` its author
 * publishes.
 */
import { existsSync, readFileSync } from 'node:fs'
import { join } from 'node:path'
import { listPluginTargets } from '../shared/plugin-quality.mjs'

const KINDS = ['mutation', 'coverage']

const readFromFolder = (dir, kind) => {
  const file = join(dir, `${kind}.json`)
  return existsSync(file) ? JSON.parse(readFileSync(file, 'utf8')) : null
}

const readFromUrl = async (base, kind, fetcher) => {
  const response = await fetcher(`${base.replace(/\/+$/, '')}/${kind}.json`)
  if (response.status === 404) return null
  if (!response.ok) throw new Error(`${kind}.json was answered with ${response.status}`)
  return response.json()
}

/**
 * The quality files at `source`, a folder or an https URL holding `mutation.json` and
 * `coverage.json`, as maps by target. A missing file is an empty map; no source is two empty maps.
 *
 * @param {string | null} source
 * @param {typeof fetch} fetcher
 * @returns {Promise<{ mutation: Map<string, object>, coverage: Map<string, object> }>}
 */
export const readQualityFiles = async (source, fetcher = fetch) => {
  const files = {}
  for (const kind of KINDS) {
    let file = null
    if (source?.startsWith('https://')) file = await readFromUrl(source, kind, fetcher)
    else if (source) file = readFromFolder(source, kind)
    files[kind] = new Map((file?.targets ?? []).map((one) => [one.target, one]))
  }
  return files
}

const getLatest = (...dates) => dates.filter(Boolean).sort().at(-1) ?? null

const hasTests = (tests) => tests.go + tests.typescript + tests.stories > 0

/**
 * The quality of the plugin folder `folder`: one entry per measured target and the counts of its
 * tests; null when nothing was measured and it holds no tests.
 *
 * @param {string} folder
 * @param {{ mutation: Map<string, object>, coverage: Map<string, object> }} files
 * @param {{ go: number, typescript: number, stories: number }} tests
 */
export const getInternalQuality = (folder, files, tests) => {
  const entries = listPluginTargets(folder).flatMap(({ target, kind }) => {
    const mutation = files.mutation.get(target)
    const coverage = files.coverage.get(target)
    if (!mutation && !coverage) return []
    return [
      {
        target,
        kind,
        mutationScore: mutation?.mutationScore ?? null,
        coverage: coverage?.coverage ?? null,
        measuredAt: getLatest(mutation?.measuredAt, coverage?.measuredAt),
      },
    ]
  })
  return entries.length === 0 && !hasTests(tests) ? null : { reportedBy: 'numen', entries, tests }
}

const readPercent = (value) =>
  typeof value === 'number' && Number.isFinite(value) && value >= 0 && value <= 100 ? value : null

const readCount = (value) => (Number.isSafeInteger(value) && value >= 0 ? value : 0)

const readTests = (tests) => ({
  go: readCount(tests?.go),
  typescript: readCount(tests?.typescript),
  stories: readCount(tests?.stories),
})

/**
 * The quality an author reports in `quality.json`; null when the file is absent or names no
 * percentage and no test count.
 */
export const getExternalQuality = (document) => {
  if (document === null || typeof document !== 'object') return null
  const mutationScore = readPercent(document.mutationScore)
  const coverage = readPercent(document.coverage)
  const tests = readTests(document.tests)
  const isMeasured = mutationScore !== null || coverage !== null
  if (!isMeasured && !hasTests(tests)) return null
  const measuredAt = typeof document.measuredAt === 'string' ? document.measuredAt : null
  return {
    reportedBy: 'author',
    entries: isMeasured ? [{ target: null, kind: null, mutationScore, coverage, measuredAt }] : [],
    tests,
  }
}
