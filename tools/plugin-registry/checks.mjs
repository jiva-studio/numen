/**
 * The checks the registry runs on every plugin itself. Each is `{ id, status, detail }`, with the
 * status `pass`, `warn`, `fail`, `info` or `skipped`; the site shows them on the Scorecard.
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

/** The permissions, and a warning when the current version added any. */
export const checkPermissions = (permissions, added) => {
  if (added.length > 0) {
    return createCheck('permissions', 'warn', `This version adds: ${added.join(', ')}.`)
  }
  return createCheck(
    'permissions',
    'info',
    permissions.length === 0 ? 'No permissions.' : `${permissions.length} permissions.`,
  )
}

/** When the plugin last changed: its last commit or its last release. */
export const checkUpdated = (date) =>
  date
    ? createCheck('updated', 'info', date.slice(0, 10))
    : createCheck('updated', 'warn', 'No commit or release date.')

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
