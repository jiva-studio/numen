/**
 * The checks the registry runs on every plugin itself. Each is `{ id, status, detail }`, with the
 * status `pass`, `warn`, `fail`, `info` or `skipped`; the site shows them on the Quality tab.
 */

/** A check result. */
export const createCheck = (id, status, detail) => ({ id, status, detail })

/** The manifest was read against the schema of `manifest.proto`; unknown keys are a warning. */
export const checkManifest = (warnings) =>
  warnings.length === 0
    ? createCheck('manifest', 'pass', 'The manifest matches the schema.')
    : createCheck('manifest', 'warn', `The schema does not name: ${warnings.join('; ')}.`)

/** The plugin's license, by SPDX id. */
export const checkLicense = (spdx) =>
  spdx
    ? createCheck('license', 'pass', spdx)
    : createCheck('license', 'fail', 'No license was found.')

/** The README is there and has an introduction before `<!-- more -->`. */
export const checkReadme = (readme) => {
  if (readme === null) return createCheck('readme', 'fail', 'No README.md.')
  if (!readme.hasIntro) return createCheck('readme', 'warn', 'The README has no <!-- more --> line.')
  return createCheck('readme', 'pass', 'README.md with an introduction.')
}

/** At least one screenshot, each in light and dark. */
export const checkScreenshots = (screenshots) => {
  if (screenshots.length === 0) return createCheck('screenshots', 'warn', 'No screenshots.')
  const unpaired = screenshots.filter((one) => one.light === null || one.dark === null)
  if (unpaired.length > 0) {
    return createCheck(
      'screenshots',
      'warn',
      `Missing light or dark: ${unpaired.map((one) => one.name).join(', ')}.`,
    )
  }
  return createCheck('screenshots', 'pass', `${screenshots.length} in light and dark.`)
}

/** A warning when the current version added permissions; null when it added none. */
export const checkAddedPermissions = (added) =>
  added.length > 0 ? createCheck('permissions', 'warn', `This version adds: ${added.join(', ')}.`) : null

/** Every release artifact has a SHA-256 digest. */
export const checkArtifacts = (artifacts) => {
  if (artifacts.length === 0) return createCheck('artifacts', 'fail', 'The release has no artifacts.')
  const missing = artifacts.filter((one) => !one.sha256)
  if (missing.length > 0) {
    return createCheck(
      'artifacts',
      'fail',
      `No SHA-256 for: ${missing.map((one) => one.name).join(', ')}.`,
    )
  }
  return createCheck('artifacts', 'pass', `${artifacts.length} artifacts with SHA-256.`)
}

const SHOWN_NAMES = 5

const listNames = (names) =>
  names.length <= SHOWN_NAMES
    ? names.join(', ')
    : `${names.slice(0, SHOWN_NAMES).join(', ')} and ${names.length - SHOWN_NAMES} more`

/** The lowest mutation score a plugin passes with. */
export const MUTATION_THRESHOLD = 80

/** The tests counted in the source, or reported by the author: a warning when there are none. */
export const checkHasTests = (tests) => {
  const total = (tests?.go ?? 0) + (tests?.typescript ?? 0) + (tests?.stories ?? 0)
  return total > 0
    ? createCheck('tests', 'pass', 'Passed')
    : createCheck('tests', 'warn', 'No tests')
}

/** Each measured mutation score is at least MUTATION_THRESHOLD percent; skipped when none is measured. */
export const checkMutationScore = (entries) => {
  const measured = (entries ?? []).filter((one) => one.mutationScore !== null)
  if (measured.length === 0) return createCheck('mutation', 'skipped', 'Not measured')
  return measured.every((one) => one.mutationScore >= MUTATION_THRESHOLD)
    ? createCheck('mutation', 'pass', 'Passed')
    : createCheck('mutation', 'warn', `Below ${MUTATION_THRESHOLD}%`)
}

/** Every Vue component of the frontend is shown by a story; skipped without a frontend or components. */
export const checkStories = (found) => {
  if (found === null) return createCheck('stories', 'skipped', 'No frontend.')
  if (found.components.length === 0) return createCheck('stories', 'skipped', 'No Vue components.')
  if (found.unstoried.length === 0) {
    return createCheck('stories', 'pass', `${found.components.length} components, each in a story.`)
  }
  const names = found.unstoried.map((one) => one.slice(one.lastIndexOf('/') + 1))
  return createCheck(
    'stories',
    'warn',
    `${found.unstoried.length} of ${found.components.length} components have no story: ${listNames(names)}.`,
  )
}

/** A page of the manual names the plugin. */
export const checkManual = (pages) => {
  if (pages === null) return createCheck('manual', 'skipped', 'No manual.')
  return pages.length > 0
    ? createCheck('manual', 'pass', pages.join('\n'))
    : createCheck('manual', 'warn', 'No page of the manual names the plugin.')
}

const describeCalls = (calls, language, title) => {
  const found = calls.filter((one) => one.language === language)
  if (found.length === 0) return []
  const files = [...new Set(found.flatMap((one) => one.files))].sort()
  return [`${title}: ${found.map((one) => one.name).join(', ')} in ${listNames(files)}.`]
}

/**
 * Whether the plugin makes network requests, as read from its code and its manifest: one line of
 * an answer, then one line per kind of evidence.
 */
export const checkNetwork = (use) => {
  const lines = [
    use.isNetworked ? 'Yes, read from the code and the manifest.' : 'None found in the code or the manifest.',
    ...describeCalls(use.calls, 'go', 'Go'),
    ...describeCalls(use.calls, 'typescript', 'Frontend'),
    ...(use.hosts.length > 0 ? [`Hosts in the code: ${use.hosts.join(', ')}.`] : []),
    ...(use.frameOrigins.length > 0 ? [`Frames: ${use.frameOrigins.join(', ')}.`] : []),
    ...(use.downloadHosts.length > 0 ? [`Downloads from: ${use.downloadHosts.join(', ')}.`] : []),
    ...(use.canOpenLinks ? ['Opens links in your browser.'] : []),
  ]
  return createCheck('network', 'info', lines.join('\n'))
}

/** No known vulnerability in the dependencies; skipped when the scan was not done. */
export const checkVulnerabilities = ({ vulnerabilities, reason }) => {
  if (vulnerabilities === null) return createCheck('vulnerabilities', 'skipped', reason)
  if (vulnerabilities.length === 0) return createCheck('vulnerabilities', 'pass', 'No known vulnerabilities.')
  const ids = [...new Set(vulnerabilities.map((one) => one.id))]
  return createCheck('vulnerabilities', 'fail', `${ids.length} known: ${listNames(ids)}.`)
}
