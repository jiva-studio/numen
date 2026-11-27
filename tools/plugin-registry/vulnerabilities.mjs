/**
 * The known vulnerabilities of a plugin's dependencies, from osv-scanner over its `go.mod` and the
 * npm lockfile. The Go standard library and the toolchain are the Go release's, not the plugin's,
 * and are left out, as are the modules and packages of this repository.
 */
import { findTool, runTool } from '../shared/tools.mjs'
import { isNumenModule, isNumenPackage } from './dependencies.mjs'

const OSV_URL = 'https://osv.dev/vulnerability/'

const GO_RELEASE_PACKAGES = new Set(['stdlib', 'toolchain'])

const MAX_BUFFER = 64 * 1024 * 1024

const isThirdParty = (pkg) => {
  if (pkg.ecosystem === 'Go') return !GO_RELEASE_PACKAGES.has(pkg.name) && !isNumenModule(pkg.name)
  if (pkg.ecosystem === 'npm') return !isNumenPackage(pkg.name)
  return true
}

const listGroups = (found) => {
  const vulnerabilities = found.vulnerabilities ?? []
  const groups = (found.groups ?? []).filter((one) => (one.ids ?? []).length > 0)
  if (groups.length === 0) return vulnerabilities.map((one) => ({ id: one.id, vulnerability: one }))
  return groups.map((group) => ({
    id: group.ids[0],
    vulnerability: vulnerabilities.find((one) => group.ids.includes(one.id)) ?? null,
  }))
}

/**
 * The vulnerabilities of an osv-scanner JSON report, one per group of aliases, by the path of the
 * file each was found in.
 *
 * @param {{ results?: object[] }} report
 * @returns {Map<string, { id: string, url: string, summary: string, ecosystem: string, name: string, version: string }[]>}
 */
export const parseOsvReport = (report) => {
  const byPath = new Map()
  for (const result of report?.results ?? []) {
    const path = result.source?.path ?? ''
    const list = byPath.get(path) ?? []
    for (const found of result.packages ?? []) {
      const pkg = found.package ?? {}
      if (!isThirdParty(pkg)) continue
      for (const { id, vulnerability } of listGroups(found)) {
        list.push({
          id,
          url: `${OSV_URL}${id}`,
          summary: vulnerability?.summary ?? '',
          ecosystem: pkg.ecosystem ?? '',
          name: pkg.name ?? '',
          version: pkg.version ?? '',
        })
      }
    }
    byPath.set(path, list)
  }
  return byPath
}

/**
 * Scans one lockfile (a `go.mod` or a `package-lock.json`) with osv-scanner. Each file is scanned
 * once. Where osv-scanner is not installed, the scan is skipped.
 *
 * @param {{ run?: typeof runTool, isInstalled?: () => boolean }} [options]
 * @returns {(file: string) => Promise<{ status: 'done', vulnerabilities: object[] } | { status: 'skipped' | 'failed', reason: string }>}
 */
export const createLockfileScanner = ({
  run = runTool,
  isInstalled = () => findTool('osv-scanner') !== null,
} = {}) => {
  const scans = new Map()
  const scan = (file) => {
    if (!isInstalled()) return { status: 'skipped', reason: 'osv-scanner is not installed.' }
    const result = run('osv-scanner', ['scan', 'source', '--lockfile', file, '--format', 'json'], {
      encoding: 'utf8',
      maxBuffer: MAX_BUFFER,
    })
    if (result.status !== 0 && result.status !== 1) {
      return { status: 'failed', reason: 'osv-scanner did not finish.' }
    }
    const report = parseOsvReport(JSON.parse(result.stdout || '{}'))
    return { status: 'done', vulnerabilities: [...report.values()].flat() }
  }
  return async (file) => {
    if (!scans.has(file)) scans.set(file, scan(file))
    return scans.get(file)
  }
}

/**
 * The vulnerabilities of a plugin: those of its `go.mod`, and those of the lockfile in the npm
 * packages it installs. Null when a scan was not done, with the reason.
 *
 * @param {(file: string) => Promise<object>} scanLockfile
 * @param {{ goMod: string | null, lockfile: string | null, npmPackages: { name: string, version: string }[] }} scope
 * @returns {Promise<{ vulnerabilities: object[] | null, reason: string }>}
 */
export const scanPluginVulnerabilities = async (scanLockfile, { goMod, lockfile, npmPackages }) => {
  const found = []
  if (goMod) {
    const scan = await scanLockfile(goMod)
    if (scan.status !== 'done') return { vulnerabilities: null, reason: scan.reason }
    found.push(...scan.vulnerabilities.filter((one) => one.ecosystem === 'Go'))
  }
  if (lockfile && npmPackages.length > 0) {
    const scan = await scanLockfile(lockfile)
    if (scan.status !== 'done') return { vulnerabilities: null, reason: scan.reason }
    const installed = new Set(npmPackages.map((one) => `${one.name}@${one.version}`))
    found.push(
      ...scan.vulnerabilities.filter(
        (one) => one.ecosystem === 'npm' && installed.has(`${one.name}@${one.version}`),
      ),
    )
  }
  return {
    vulnerabilities: found.sort((a, b) => a.name.localeCompare(b.name) || a.id.localeCompare(b.id)),
    reason: '',
  }
}
