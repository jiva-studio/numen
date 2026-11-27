/**
 * The third-party dependencies of a plugin: the Go modules its `go.mod` requires and the npm
 * packages of the `dependencies` of its `package.json`, with their versions and licenses. Modules
 * and packages of this repository (`github.com/jiva-studio/numen/...`, `@numen/*`) are the Numen SDK
 * and are left out.
 */
import { mkdtempSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { findTool, runTool } from '../shared/tools.mjs'

const NUMEN_GO_PREFIX = 'github.com/jiva-studio/numen/'

const NUMEN_NPM_PREFIX = '@numen/'

const REQUIRE_LINE = /^\s*([^\s()]+)\s+(v[^\s]+)(\s*\/\/\s*indirect)?\s*$/

/** Whether a Go module path belongs to this repository. */
export const isNumenModule = (path) => path.startsWith(NUMEN_GO_PREFIX)

/** Whether an npm package name belongs to this repository. */
export const isNumenPackage = (name) => name.startsWith(NUMEN_NPM_PREFIX)

/**
 * The third-party requirements of a `go.mod`, in the order the file lists them.
 *
 * @param {string} text - the go.mod
 * @returns {{ name: string, version: string, isIndirect: boolean }[]}
 */
export const parseGoMod = (text) => {
  const found = []
  let isInBlock = false
  for (const raw of text.split('\n')) {
    const line = raw.trim()
    if (isInBlock && line === ')') {
      isInBlock = false
      continue
    }
    if (line === 'require (') {
      isInBlock = true
      continue
    }
    const body = isInBlock ? line : line.startsWith('require ') ? line.slice('require '.length) : null
    const match = body === null ? null : REQUIRE_LINE.exec(body)
    if (!match || isNumenModule(match[1])) continue
    found.push({ name: match[1], version: match[2], isIndirect: Boolean(match[3]) })
  }
  return found
}

const getLockKey = (name) => `node_modules/${name}`

const resolveLockPackage = (lock, from, name) => {
  for (let base = from; ; ) {
    const key = base === '' ? getLockKey(name) : `${base}/${getLockKey(name)}`
    if (lock.packages?.[key]) return key
    if (base === '') return null
    const at = base.lastIndexOf('/node_modules/')
    base = at === -1 ? '' : base.slice(0, at)
  }
}

/**
 * The third-party npm packages of `dependencies` in a package.json, each at the version and with
 * the license the lockfile resolves it to; a package the lockfile does not hold keeps its declared
 * range and has no license.
 *
 * @param {{ dependencies?: Record<string, string> } | null} pkg
 * @param {{ packages?: Record<string, { version?: string, license?: string }> } | null} lock
 * @returns {{ name: string, version: string, license: string | null }[]}
 */
export const listNpmDependencies = (pkg, lock) =>
  Object.entries(pkg?.dependencies ?? {})
    .filter(([name]) => !isNumenPackage(name))
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([name, range]) => {
      const entry = lock?.packages?.[getLockKey(name)]
      return {
        name,
        version: entry?.version ?? range,
        license: typeof entry?.license === 'string' ? entry.license : null,
      }
    })

/**
 * Every package the lockfile installs for `names`: the packages themselves and, transitively, the
 * `dependencies` and `optionalDependencies` of each, resolved as Node resolves them.
 *
 * @param {{ packages?: Record<string, object> } | null} lock
 * @param {string[]} names - top-level package names
 * @returns {{ name: string, version: string }[]} ordered by name, then version
 */
export const listNpmPackages = (lock, names) => {
  if (!lock?.packages) return []
  const seen = new Map()
  const pending = names.map((name) => resolveLockPackage(lock, '', name)).filter(Boolean)
  while (pending.length > 0) {
    const key = pending.pop()
    if (seen.has(key)) continue
    const entry = lock.packages[key]
    const name = entry.name ?? key.slice(key.lastIndexOf('node_modules/') + 'node_modules/'.length)
    seen.set(key, { name, version: entry.version ?? '' })
    const children = { ...entry.dependencies, ...entry.optionalDependencies }
    for (const child of Object.keys(children)) {
      const resolved = resolveLockPackage(lock, key, child)
      if (resolved && !seen.has(resolved)) pending.push(resolved)
    }
  }
  return [...seen.values()].sort(
    (a, b) => a.name.localeCompare(b.name) || a.version.localeCompare(b.version),
  )
}

/** The key a Go module version is known by. */
export const getModuleKey = (module) => `${module.name}@${module.version}`

const parseJsonStream = (text) => {
  const trimmed = text.trim()
  return trimmed === '' ? [] : JSON.parse(`[${trimmed.replace(/\}\s*\{/g, '},{')}]`)
}

const LICENSES_HELPER = join('tools', 'plugin-registry', 'licenses')

const MAX_BUFFER = 64 * 1024 * 1024

/**
 * Reads the licenses of Go module versions: `go mod download -json` puts each in the module cache,
 * offline with GOPROXY=off so only the cache answers, and `tools/plugin-registry/licenses` reads
 * its license files, built on its own go.mod. A module that cannot be downloaded, and every module where Go is not
 * installed, has no license.
 *
 * @param {{ root: string, isOffline: boolean }} options
 * @returns {(modules: { name: string, version: string }[]) => Promise<Map<string, string | null>>}
 *   licenses by `getModuleKey`, SPDX ids joined with ", "
 */
export const createGoLicenseReader = ({ root, isOffline }) => {
  const known = new Map()
  return async (modules) => {
    const missing = [...new Set(modules.map(getModuleKey))].filter((key) => !known.has(key))
    if (missing.length > 0 && findTool('go') === null) {
      for (const key of missing) known.set(key, null)
    }
    const asked = missing.filter((key) => !known.has(key))
    if (asked.length > 0) {
      const work = mkdtempSync(join(tmpdir(), 'plugin-go-modules-'))
      try {
        const env = {
          ...process.env,
          GOWORK: 'off',
          GOFLAGS: '-mod=mod',
          ...(isOffline ? { GOPROXY: 'off' } : {}),
        }
        const download = runTool('go', ['mod', 'download', '-json', ...asked], {
          cwd: work,
          env,
          encoding: 'utf8',
          maxBuffer: MAX_BUFFER,
        })
        const dirs = new Map(
          parseJsonStream(download.stdout ?? '')
            .filter((one) => one.Dir && !one.Error)
            .map((one) => [`${one.Path}@${one.Version}`, one.Dir]),
        )
        const read = dirs.size === 0
          ? { status: 0, stdout: '{}' }
          : runTool('go', ['run', '.', ...dirs.values()], {
              cwd: join(root, LICENSES_HELPER),
              env: { ...process.env, GOWORK: 'off' },
              encoding: 'utf8',
              maxBuffer: MAX_BUFFER,
            })
        const ids = read.status === 0 ? JSON.parse(read.stdout) : {}
        for (const key of asked) {
          const found = ids[dirs.get(key)] ?? []
          known.set(key, found.length > 0 ? found.join(', ') : null)
        }
      } finally {
        rmSync(work, { recursive: true, force: true })
      }
    }
    return new Map(modules.map((one) => [getModuleKey(one), known.get(getModuleKey(one)) ?? null]))
  }
}
