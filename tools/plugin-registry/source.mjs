/**
 * What the registry reads from the source folder of a plugin, the same for a plugin of this
 * repository and a clone of another: its third-party dependencies with their licenses, their known
 * vulnerabilities, its network use and the Vue components no story shows.
 */
import { existsSync, readFileSync } from 'node:fs'
import { join } from 'node:path'
import {
  createGoLicenseReader,
  getModuleKey,
  listNpmDependencies,
  listNpmPackages,
  parseGoMod,
} from './dependencies.mjs'
import { readNetworkUse } from './network.mjs'
import { listUnstoriedComponents } from './tests.mjs'
import { createLockfileScanner, scanPluginVulnerabilities } from './vulnerabilities.mjs'

/** The npm lockfile of a repository root. */
export const LOCKFILE = 'package-lock.json'

const readJson = (file) => (existsSync(file) ? JSON.parse(readFileSync(file, 'utf8')) : null)

/**
 * The readers that call outside tools; a test passes its own.
 *
 * @param {{ root: string, isOffline: boolean }} options
 * @returns {{ isOffline: boolean, readGoLicenses: ReturnType<typeof createGoLicenseReader>, scanLockfile: ReturnType<typeof createLockfileScanner> }}
 */
export const createSourceReaders = ({ root, isOffline }) => ({
  isOffline,
  readGoLicenses: createGoLicenseReader({ root, isOffline }),
  scanLockfile: createLockfileScanner(),
})

const OFFLINE_SCAN = { vulnerabilities: null, reason: 'Not run offline.' }

/**
 * The source facts of the plugin folder `dir`, whose npm packages the lockfile of `lockRoot`
 * resolves.
 *
 * @param {object} options
 * @param {string} options.dir - the plugin folder
 * @param {string} options.lockRoot - the folder that holds package-lock.json
 * @param {Record<string, any>} options.json - the manifest in the schema's JSON form
 * @param {ReturnType<typeof createSourceReaders>} options.readers
 */
export const inspectSource = async ({ dir, lockRoot, json, readers }) => {
  const goModFile = join(dir, 'go.mod')
  const hasGoMod = existsSync(goModFile)
  const requires = hasGoMod ? parseGoMod(readFileSync(goModFile, 'utf8')) : []
  const direct = requires.filter((one) => !one.isIndirect)
  const licenses = await readers.readGoLicenses(direct)
  const lockfile = join(lockRoot, LOCKFILE)
  const lock = readJson(lockfile)
  const npm = listNpmDependencies(readJson(join(dir, 'package.json')), lock)
  const directNames = new Set(npm.map((one) => one.name))
  const npmPackages = listNpmPackages(lock, [...directNames])
  const dependencies = {
    packages: [
      ...direct.map((one) => ({
        ecosystem: 'go',
        name: one.name,
        version: one.version,
        license: licenses.get(getModuleKey(one)) ?? null,
      })),
      ...npm.map((one) => ({ ecosystem: 'npm', ...one })),
    ],
    indirect: {
      go: requires.length - direct.length,
      npm: npmPackages.filter((one) => !directNames.has(one.name)).length,
    },
  }
  const scan = readers.isOffline
    ? OFFLINE_SCAN
    : await scanPluginVulnerabilities(readers.scanLockfile, {
        goMod: hasGoMod && requires.length > 0 ? goModFile : null,
        lockfile: lock ? lockfile : null,
        npmPackages,
      })
  return {
    dependencies,
    scan,
    network: readNetworkUse(dir, json),
    stories: listUnstoriedComponents(dir),
  }
}
