/**
 * The per-plugin quality files: the mutation score of `.github/workflows/mutation.yml` and the
 * statement coverage of `.github/workflows/coverage.yml`, one entry per measured target.
 *
 * A file is `{ "measuredAt": "<ISO time>", "targets": [{ "target", "mutationScore" | "coverage",
 * "measuredAt" }] }`, ordered by target. A target is named as `tools/ci-mutation/targets.mjs`
 * names it: `plugins-<folder>`, with `-go` or `-ts` where the folder holds both.
 */

/** The storage paths of the quality files, under the downloads zone. */
export const QUALITY_FILES = {
  mutation: 'plugins/quality/mutation.json',
  coverage: 'plugins/quality/coverage.json',
}

const TARGET_PREFIX = 'plugins-'
const KIND_SUFFIXES = ['', '-go', '-ts']

/** Whether a target is a plugin folder. */
export const isPluginTarget = (name) => name.startsWith(TARGET_PREFIX)

/** The target names a plugin folder can be measured under, with the kind each one names. */
export const listPluginTargets = (folder) =>
  KIND_SUFFIXES.map((suffix) => ({
    target: `${TARGET_PREFIX}${folder}${suffix}`,
    kind: suffix === '' ? null : suffix.slice(1),
  }))

/** A value rounded to one decimal place. */
export const roundScore = (value) => Math.round(value * 10) / 10

/**
 * The entries of `previous` with `entries` laid over them by target: a target measured now takes
 * its new entry, a target not measured now keeps the one it had.
 *
 * @param {{ targets?: object[] } | null} previous - the file as stored, or null
 * @param {{ target: string }[]} entries - the entries measured now
 * @param {string} measuredAt - the time of this measurement
 * @returns {{ measuredAt: string, targets: object[] }} the file to store
 */
export const mergeQualityFile = (previous, entries, measuredAt) => {
  const byTarget = new Map((previous?.targets ?? []).map((one) => [one.target, one]))
  for (const entry of entries) byTarget.set(entry.target, entry)
  return {
    measuredAt,
    targets: [...byTarget.values()].sort((a, b) => a.target.localeCompare(b.target)),
  }
}

/** The text a quality file holds. */
export const formatQualityFile = (file) => `${JSON.stringify(file, null, 2)}\n`
