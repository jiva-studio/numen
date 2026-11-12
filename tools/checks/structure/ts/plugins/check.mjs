import { spawnSync } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'
import { parse as parseYaml } from 'yaml'
import { listPluginNames, readPluginManifests } from '../../../../shared/manifests.mjs'
import { findPluginCommandMismatches } from '../../../../shared/registrations.mjs'
import { createDeterminismSelectors } from '../../../../lint-layers/base-eslint.mjs'
import {
  GO_MODULE_ROOT,
  MODULES_DIR,
  REPO_ROOT,
  failOnFindings,
  getGoModReferences,
  getGoReferences,
  getModuleSpecifiers,
  parsePackageJson,
  listShippedFiles,
  readSource,
  requireMatches,
} from '../../../lib/graph.mjs'

const CORE_MODULE = `${GO_MODULE_ROOT}libs/core/kernel`

export function verifyPluginsStatic() {
  const pluginsDir = path.join(MODULES_DIR, 'plugins')
  readPluginManifests(pluginsDir)

  const commandMismatches = findPluginCommandMismatches(pluginsDir)
  if (commandMismatches.length > 0) {
    throw new Error(
      'Plugin command declaration and handler mismatch:\n' + commandMismatches.join('\n'),
    )
  }

  if (fs.existsSync(path.join(pluginsDir, 'shared'))) {
    throw new Error('plugins/shared exists: a plugin shares code only through the SDK')
  }

  const leaks = []
  for (const file of listShippedFiles(['plugins'])) {
    if (!file.endsWith('.go') && path.basename(file) !== 'go.mod') continue
    const text = fs.readFileSync(path.join(REPO_ROOT, file), 'utf8')
    const { imports } = file.endsWith('.go') ? getGoReferences(text) : getGoModReferences(text)
    for (const { line, value } of imports) {
      if (value === CORE_MODULE || value.startsWith(`${CORE_MODULE}/`)) {
        leaks.push({ file, line, symbol: value })
      }
    }
  }
  failOnFindings('Plugin imports core package', leaks)
}

/** The packages a plugin depends on; their `package.json` `exports` declare the entries a plugin may name. */
const PLUGIN_PACKAGES = ['@numen/plugin-sdk', '@numen/ui']

/** A build configuration file, which may name the build entries a package declares. */
const BUILD_CONFIGURATION = /\.config\.[cm]?[jt]s$/

/** The entry a plugin file may not name: `testing` is the test entry a package declares, a stylesheet is an asset. */
const TEST_ENTRY = './testing'

/**
 * Lists the specifiers of `files` that reach into a plugin package beyond its main entry. `files`
 * holds `{ file, literals }` with each literal as `{ line, value }`, and `exportsByPackage` maps a
 * package name to the keys and targets of its `exports`. A deep path passes when the package
 * declares it and it is the test entry, a stylesheet, or named from a build configuration file.
 */
export function findPluginEntryViolations(files, exportsByPackage) {
  const found = []
  for (const { file, literals } of files) {
    for (const { line, value } of literals) {
      const name = PLUGIN_PACKAGES.find((one) => value.startsWith(one + '/'))
      if (!name) continue
      const entry = '.' + value.slice(name.length)
      const target = exportsByPackage[name]?.[entry]
      if (target === undefined) {
        found.push({
          file,
          line,
          symbol: `${JSON.stringify(value)} is not an entry ${name} declares`,
        })
        continue
      }
      const targetPath = typeof target === 'string' ? target : JSON.stringify(target)
      const isAllowed =
        entry === TEST_ENTRY ||
        /\.css\b/.test(targetPath) ||
        BUILD_CONFIGURATION.test(path.basename(file))
      if (!isAllowed)
        found.push({
          file,
          line,
          symbol: `${JSON.stringify(value)} reaches past the main entry of ${name}`,
        })
    }
  }
  return found
}

/** The `exports` map of each plugin package, read from the `package.json` under `libs/` that names it. */
export function readPluginPackageExports() {
  const result = {}
  for (const file of listShippedFiles(['libs']).filter(
    (one) => path.basename(one) === 'package.json' && !one.includes('node_modules/'),
  )) {
    const manifest = parsePackageJson(fs.readFileSync(path.join(REPO_ROOT, file), 'utf8'))
    if (PLUGIN_PACKAGES.includes(manifest.name)) result[manifest.name] = manifest.exports ?? {}
  }
  return result
}

/** A plugin imports `@numen/plugin-sdk` and `@numen/ui` through their main entry, and from the test entry, stylesheets and build entries a package declares. */
export function verifyPluginsUseMainEntry() {
  const exportsByPackage = readPluginPackageExports()
  for (const name of PLUGIN_PACKAGES)
    requireMatches(`exports of ${name}`, Object.keys(exportsByPackage[name] ?? {}).length)
  const files = []
  let named = 0
  for (const file of listShippedFiles(['plugins'])) {
    if (!/\.(ts|tsx|mts|cts|js|mjs|cjs|vue)$/.test(file)) continue
    const source = readSource(file)
    if (!source) continue
    const literals = source.literals.filter(({ value }) =>
      PLUGIN_PACKAGES.some((one) => value === one || value.startsWith(one + '/')),
    )
    named += literals.length
    files.push({ file, literals })
  }
  requireMatches('plugin references to the plugin packages', named)
  failOnFindings(
    'A plugin reaches past the main entry of @numen/plugin-sdk or @numen/ui',
    findPluginEntryViolations(files, exportsByPackage),
  )
}

const OUTER_BACKEND_PACKAGES = ['service', 'engine']
const WIRE_BACKEND_PACKAGE = 'wire'
const NON_INNER_BACKEND_PACKAGES = ['cmd', ...OUTER_BACKEND_PACKAGES, WIRE_BACKEND_PACKAGE, 'testsupport']

const isBackendPackageImport = (value, name) => value.endsWith(`/backend/${name}`) || value.includes(`/backend/${name}/`)

/**
 * Finds the imports that run against the layers of a plugin backend: an inner package (below `service` and `wire`)
 * imports `service`, `engine` or `wire`; a `wire` package imports `service`, `engine` or an adapter. `wire` is the layer between
 * `service` and `api`, the domain, the use cases and the ports: the service imports it, it imports them, nothing below
 * imports it. `files` holds `{ file, imports }` with each import as `{ line, value }`.
 */
export function findPluginBackendLayerInversions(files) {
  const found = []
  for (const { file, imports } of files) {
    const [root, plugin, tier, subpkg] = file.split('/')
    if (root !== 'plugins' || tier !== 'backend' || !subpkg || subpkg.endsWith('.go')) continue
    const isWire = subpkg === WIRE_BACKEND_PACKAGE
    if (!isWire && NON_INNER_BACKEND_PACKAGES.includes(subpkg)) continue
    const forbidden = isWire
      ? (value) => OUTER_BACKEND_PACKAGES.some((outer) => isBackendPackageImport(value, outer)) || value.includes('/backend/internal/adapter/')
      : (value) => [...OUTER_BACKEND_PACKAGES, WIRE_BACKEND_PACKAGE].some((outer) => isBackendPackageImport(value, outer))
    for (const { line, value } of imports) {
      if (forbidden(value)) {
        found.push({
          file,
          line,
          symbol: `${isWire ? 'wire' : 'inner'} package backend/${subpkg} imports ${value} in plugins/${plugin}, against the direction service, wire, then api, domain, usecase and ports`,
        })
      }
    }
  }
  return found
}

export function verifyPluginBackendInternalLayers() {
  const files = listShippedFiles(['plugins/*.go']).map((file) => ({
    file,
    imports: getGoReferences(fs.readFileSync(path.join(REPO_ROOT, file), 'utf8')).imports,
  }))
  failOnFindings(
    'Plugin backend package imports against the layers: inner package imports service, engine or wire; wire imports service, engine or an adapter',
    findPluginBackendLayerInversions(files),
  )
}

export const MAX_SERVICE_LINES = 600
const REQUIRED_INNER_KINDS = ['domain', 'usecase', 'ports']

/**
 * The service package of a plugin backend is the composition root and the wire. Finds a service
 * whose non-test code is longer than the limit, and says which of the domain, use case and ports
 * packages the backend lacks to move that code into.
 *
 * @param {{ plugin: string, serviceLines: number, innerPackages: string[] }[]} backends
 */
export function findOversizedServices(backends) {
  const found = []
  for (const { plugin, serviceLines, innerPackages } of backends) {
    if (serviceLines <= MAX_SERVICE_LINES) continue
    const missing = REQUIRED_INNER_KINDS.filter(
      (kind) => !innerPackages.some((name) => name === kind || name.startsWith(`${kind}/`)),
    )
    const lacking = missing.length > 0 ? `; the backend has no ${missing.join(', ')} package` : ''
    found.push({
      file: `plugins/${plugin}/backend/service`,
      line: 1,
      symbol: `service holds ${serviceLines} non-test lines, at most ${MAX_SERVICE_LINES}${lacking}`,
    })
  }
  return found
}

/**
 * The standard library packages a service may import: they compute on values and reach no machine, process, file,
 * network or shared state. `net/http` is there for the status codes and the handler type of the SDK server, and
 * `time` for its types; a service that needs the clock takes it from the SDK. `sync`, `os`, `log` and the rest are not
 * on the list.
 */
export const SERVICE_STANDARD_IMPORTS = [
  'bytes',
  'cmp',
  'context',
  'encoding/json',
  'errors',
  'fmt',
  'maps',
  'math',
  'net/http',
  'path',
  'regexp',
  'slices',
  'sort',
  'strconv',
  'strings',
  'time',
  'unicode',
  'unicode/utf8',
]

/** The folder of the Go SDK in the repository. */
const SDK_GO_DIR = 'libs/plugin/sdk-go'

/** The SDK packages a plugin service may import: every package of the Go SDK. */
const SDK_GO_MODULE = `${GO_MODULE_ROOT}${SDK_GO_DIR}`

/** The packages of the plugin's own backend a service may import: its API, domain, use cases, ports and wire. */
export const SERVICE_OWN_PACKAGES = ['api', 'domain', 'usecase', 'ports', 'wire']

/** The folder of the plugin's own adapters: the service is the composition root and builds them. */
const SERVICE_OWN_ADAPTERS = 'internal/adapter/'

/** Whether a service of `plugin` may import `value`: a listed standard package, an SDK package or an own package of the list. */
function isAllowedServiceImport(value, plugin) {
  if (SERVICE_STANDARD_IMPORTS.includes(value)) return true
  if (value === SDK_GO_MODULE || value.startsWith(`${SDK_GO_MODULE}/`)) return true
  const own = `${GO_MODULE_ROOT}plugins/${plugin}/backend/`
  if (!value.startsWith(own)) return false
  const inner = value.slice(own.length)
  return inner.startsWith(SERVICE_OWN_ADAPTERS) || SERVICE_OWN_PACKAGES.includes(inner.split('/')[0])
}

/**
 * Finds the imports of a backend `service` package that are not on the allow list: the listed standard packages, the
 * packages of the Go SDK, the `api`, `domain`, `usecase`, `ports` and `wire` packages of its own plugin and its own
 * `internal/adapter/*` packages. Machine access (`os`, `sync`, `net`), another standard package, a package of another
 * plugin and any other package of its own backend are refused. `files` holds `{ file, imports }` with each import as `{ line, value }`.
 */
export function findServiceImportsOffAllowList(files) {
  const found = []
  for (const { file, imports } of files) {
    const plugin = file.split('/')[1]
    for (const { line, value } of imports) {
      if (!isAllowedServiceImport(value, plugin)) {
        found.push({
          file,
          line,
          symbol: `service imports ${value}, which is not on the allow list; machine access, state and own packages beyond api, domain, usecase, ports, wire and internal/adapter belong behind a use case`,
        })
      }
    }
  }
  return found
}

function listServiceSources() {
  return listShippedFiles(['plugins/*/backend/service/*.go'])
    .filter((file) => /^plugins\/[^/]+\/backend\/service\/[^/]+\.go$/.test(file) && !file.endsWith('_test.go'))
    .map((file) => ({
      file,
      imports: getGoReferences(fs.readFileSync(path.join(REPO_ROOT, file), 'utf8')).imports,
    }))
}

export function verifyPluginServiceImports() {
  const files = listServiceSources()
  requireMatches('plugin service source files', files.length)
  failOnFindings(
    'Plugin backend service package imports a package off the allow list',
    findServiceImportsOffAllowList(files),
  )
}

/** The longest production file of a Go package and the longest package, as the Go source-size check sets them. */
export const MAX_FILE_LINES = 500
export const MAX_PACKAGE_LINES = 3000

/** The lines of a source text, counted as the Go source-size check counts them. */
function countLines(text) {
  return text.split('\n').length - (text.endsWith('\n') || text === '' ? 1 : 0)
}

/**
 * Finds the files and packages of plugin backends that are too long: a production file over `MAX_FILE_LINES`, a package
 * (a directory) over `MAX_PACKAGE_LINES`. Tests are not counted. The `service` package has its own, smaller limit.
 *
 * @param {{ file: string, lines: number }[]} files The production Go files under `plugins/<name>/backend`
 */
export function findOversizedBackendPackages(files) {
  const found = []
  const byDir = new Map()
  for (const { file, lines } of files) {
    if (lines > MAX_FILE_LINES) {
      found.push({ file, line: MAX_FILE_LINES + 1, symbol: `file holds ${lines} lines, at most ${MAX_FILE_LINES}` })
    }
    const dir = path.posix.dirname(file)
    byDir.set(dir, (byDir.get(dir) ?? 0) + lines)
  }
  for (const [dir, lines] of byDir) {
    if (lines > MAX_PACKAGE_LINES) {
      found.push({ file: dir, line: 1, symbol: `package holds ${lines} non-test lines, at most ${MAX_PACKAGE_LINES}` })
    }
  }
  return found
}

function listBackendFiles() {
  return listShippedFiles(['plugins/*/backend/**/*.go', 'plugins/*/backend/*.go'])
    .filter((file) => /^plugins\/[^/]+\/backend\//.test(file) && !file.endsWith('_test.go'))
    .map((file) => ({ file, lines: countLines(fs.readFileSync(path.join(REPO_ROOT, file), 'utf8')) }))
}

function listPluginBackends(files) {
  const byPlugin = new Map()
  for (const { file, lines } of files) {
    const [, plugin, , ...rest] = file.split('/')
    const entry = byPlugin.get(plugin) ?? { plugin, serviceLines: 0, innerPackages: new Set() }
    byPlugin.set(plugin, entry)
    if (rest.length < 2) continue
    if (rest[0] === 'service') {
      entry.hasService = true
      entry.serviceLines += lines
    } else {
      entry.innerPackages.add(rest.slice(0, -1).join('/'))
    }
  }
  return [...byPlugin.values()]
    .filter((entry) => entry.hasService)
    .map((entry) => ({ ...entry, innerPackages: [...entry.innerPackages] }))
}

export function verifyPluginServiceSize() {
  const files = listBackendFiles()
  const backends = listPluginBackends(files)
  requireMatches('plugin backends with a service package', backends.length)
  failOnFindings('Plugin backend package is too large', [
    ...findOversizedServices(backends),
    ...findOversizedBackendPackages(files),
  ])
}

/**
 * Golden Template & Config Drift Verification for Standalone Plugins.
 */
export function parseEnabledGoLinters(yamlText) {
  return [...(parseYaml(yamlText)?.linters?.enable ?? [])].sort((a, b) => a.localeCompare(b))
}

function findPluginConfigDrift() {
  const found = []
  const exampleDir = path.join(REPO_ROOT, 'plugins/.example')
  const rootGolangci = path.join(REPO_ROOT, '.golangci.yml')
  const baseEslint = path.join(REPO_ROOT, 'tools/lint-layers/base-eslint.mjs')
  const exampleEslint = path.join(exampleDir, 'eslint.config.js')

  for (const dir of ['plugins/.example', SDK_GO_DIR]) {
    const golangci = path.join(REPO_ROOT, dir, '.golangci.yml')
    if (!fs.existsSync(rootGolangci) || !fs.existsSync(golangci)) continue
    const rootLinters = parseEnabledGoLinters(fs.readFileSync(rootGolangci, 'utf8'))
    const linters = parseEnabledGoLinters(fs.readFileSync(golangci, 'utf8'))
    for (const l of rootLinters) {
      if (!linters.includes(l)) {
        found.push({
          file: `${dir}/.golangci.yml`,
          line: 1,
          symbol: `missing linter '${l}' enabled in root .golangci.yml`,
        })
      }
    }
    for (const l of linters) {
      if (!rootLinters.includes(l)) {
        found.push({
          file: `${dir}/.golangci.yml`,
          line: 1,
          symbol: `extra linter '${l}' not in root .golangci.yml`,
        })
      }
    }
  }

  if (fs.existsSync(baseEslint) && fs.existsSync(exampleEslint)) {
    const exText = fs.readFileSync(exampleEslint, 'utf8')
    const literals = new Set(
      readSource('plugins/.example/eslint.config.js')?.literals.map((one) => one.value) ?? [],
    )
    const requiredSelectors = createDeterminismSelectors()
      .map((one) => one.selector)
      .filter((selector) => !selector.includes('performance'))
    for (const selector of [
      ...requiredSelectors,
      '**/lib/**/*.ts',
      '**/model/**/*.ts',
      '**/api/wire*',
    ]) {
      if (!literals.has(selector)) {
        found.push({
          file: 'plugins/.example/eslint.config.js',
          line: 1,
          symbol: `missing rule selector '${selector}' from base-eslint.mjs`,
        })
      }
    }
    const specifiers = getModuleSpecifiers(exampleEslint, exText)
    // A relative import names a file beside the configuration, which every plugin copies with it.
    const isBeside = (s) => path.dirname(path.resolve(path.dirname(exampleEslint), s)) === path.dirname(exampleEslint)
    if (specifiers.some((s) => (s.startsWith('.') && !isBeside(s)) || s.startsWith('@numen/'))) {
      found.push({
        file: 'plugins/.example/eslint.config.js',
        line: 1,
        symbol:
          'plugins/.example/eslint.config.js must be 100% self-contained without ../ paths or SDK coupling',
      })
    }
  }

  const pluginsDir = path.join(REPO_ROOT, 'plugins')
  const pluginNames = listPluginNames(pluginsDir)
  const pluginCopies = [
    'plugins/.example',
    ...pluginNames
      .filter((name) => fs.existsSync(path.join(pluginsDir, name, 'go.mod')))
      .map((name) => `plugins/${name}`),
  ]
  // The SDK has a .golangci.yml of its own, whose linters findPluginConfigDrift compares with the root.
  const copiesByName = {
    '.golangci.yml': pluginCopies,
    '.golangci-structure.yml': [...pluginCopies, SDK_GO_DIR],
  }
  for (const [name, goConfigCopies] of Object.entries(copiesByName)) {
    const rootFile = path.join(REPO_ROOT, name)
    if (!fs.existsSync(rootFile)) {
      found.push({ file: name, line: 1, symbol: 'the root Go lint configuration is missing' })
      continue
    }
    const rootText = fs.readFileSync(rootFile, 'utf8')
    for (const dir of goConfigCopies) {
      const copy = path.join(REPO_ROOT, dir, name)
      if (!fs.existsSync(copy) || fs.readFileSync(copy, 'utf8') !== rootText) {
        found.push({
          file: `${dir}/${name}`,
          line: 1,
          symbol: `must be a byte-identical copy of the root ${name}`,
        })
      }
    }
  }

  for (const parentConfig of ['plugins/.golangci.yml', 'plugins/eslint.config.js']) {
    if (fs.existsSync(path.join(REPO_ROOT, parentConfig))) {
      found.push({
        file: parentConfig,
        line: 1,
        symbol:
          'shared parent config outside plugin directory (each plugin must carry its own copy of plugins/.example/)',
      })
    }
  }

  const rulesSource = path.join(REPO_ROOT, 'tools/lint-layers/numen-rules.mjs')
  const rulesGolden = path.join(exampleDir, 'numen-rules.mjs')
  if (!fs.existsSync(rulesGolden) || fs.readFileSync(rulesSource, 'utf8') !== fs.readFileSync(rulesGolden, 'utf8')) {
    found.push({
      file: 'plugins/.example/numen-rules.mjs',
      line: 1,
      symbol: 'golden rules drifted from tools/lint-layers/numen-rules.mjs (copy the file with cp)',
    })
  }

  const checkGoldenFile = (plugin, fileName) => {
    const goldenPath = path.join(exampleDir, fileName)
    const targetPath = path.join(pluginsDir, plugin, fileName)
    const relTarget = `plugins/${plugin}/${fileName}`
    if (!fs.existsSync(goldenPath)) return
    if (!fs.existsSync(targetPath)) {
      found.push({
        file: relTarget,
        line: 1,
        symbol: `missing standalone config (must match plugins/.example/${fileName})`,
      })
      return
    }
    const goldenText = fs.readFileSync(goldenPath, 'utf8')
    const targetText = fs.readFileSync(targetPath, 'utf8')
    if (goldenText !== targetText) {
      found.push({
        file: relTarget,
        line: 1,
        symbol: `config drifted from golden template plugins/.example/${fileName}`,
      })
    }
  }

  for (const plugin of pluginNames) {
    const pDir = path.join(pluginsDir, plugin)
    checkGoldenFile(plugin, 'Makefile')
    if (fs.existsSync(path.join(pDir, 'go.mod'))) {
      checkGoldenFile(plugin, '.golangci.yml')
    }
    if (fs.existsSync(path.join(pDir, 'package.json'))) {
      checkGoldenFile(plugin, 'eslint.config.js')
      checkGoldenFile(plugin, 'numen-rules.mjs')
      checkGoldenFile(plugin, 'tsconfig.json')
    }
  }

  return found
}

export function verifyPluginConfigGoldenParity() {
  failOnFindings(
    'Plugin configuration missing or drifted from golden template plugins/.example/',
    findPluginConfigDrift(),
  )
}

/** Runs ESLint across each frontend plugin directory using its own `eslint.config.js`. */
export function verifyPluginsFrontend() {
  const pluginsDir = path.join(REPO_ROOT, 'plugins')
  const exampleConfig = path.join(pluginsDir, '.example', 'eslint.config.js')
  if (!fs.existsSync(exampleConfig)) {
    throw new Error('Missing golden frontend config: plugins/.example/eslint.config.js')
  }
  const eslintBin = path.join(REPO_ROOT, 'node_modules/.bin/eslint')
  const frontendPlugins = listPluginNames(pluginsDir).filter((name) =>
    fs.existsSync(path.join(pluginsDir, name, 'frontend')),
  )

  const failures = []
  for (const plugin of frontendPlugins) {
    const pluginDir = path.join(pluginsDir, plugin)
    const configPath = path.join(pluginDir, 'eslint.config.js')
    if (!fs.existsSync(configPath)) {
      failures.push(`plugins/${plugin}: missing eslint.config.js`)
      continue
    }
    const res = spawnSync(eslintBin, ['.'], { cwd: pluginDir, encoding: 'utf8' })
    if (res.status !== 0) {
      failures.push(`plugins/${plugin}:\n${((res.stdout || '') + (res.stderr || '')).trim()}`)
    }
  }
  if (failures.length > 0) {
    throw new Error(failures.join('\n\n'))
  }
}

/** The files at the root of a plugin frontend: the entry, the words and types the slices share, and the stylesheet. */
const FRONTEND_ROOT_FILES = ['index.ts', 'words.ts', 'types.ts', 'styles.css']

/** The folders at the root of a frontend that belong to no slice: the shared `api` and `lib`, and the `ui` and `model` that compose the slices. */
const FRONTEND_ROOT_SEGMENTS = ['api', 'lib', 'model', 'ui']

/** The segments of a slice. */
const SLICE_SEGMENTS = ['ui', 'model', 'api', 'lib']

/** The folders that hold test support at any depth; their files are not shipped. */
const SUPPORT_FOLDERS = ['fixtures', 'testdata', 'testing']

const FRONTEND_FILE = /^plugins\/([^/]+)\/frontend\/(.+)$/

/** The frontend files of `files`, as `{ file, plugin, parts }` with the path below `frontend/` split; tests, stories and test support are left out. */
function listShippedFrontendFiles(files) {
  const found = []
  for (const file of files) {
    const match = FRONTEND_FILE.exec(file)
    if (!match) continue
    const parts = match[2].split('/')
    if (/\.(test|stories)\.ts$/.test(file) || parts.some((one) => SUPPORT_FOLDERS.includes(one))) continue
    found.push({ file, plugin: match[1], parts })
  }
  return found
}

/** The slices of the plugin frontends in `files`: every folder at the root of a frontend that is none of the shared or composing segments. */
export function listPluginSlices(files) {
  const slices = new Set()
  for (const { plugin, parts } of listShippedFrontendFiles(files)) {
    if (parts.length > 1 && !FRONTEND_ROOT_SEGMENTS.includes(parts[0])) slices.add(`${plugin}/${parts[0]}`)
  }
  return [...slices].sort()
}

/**
 * Lists what breaks the shape of a plugin frontend: a file at its root besides the entry, the words, the
 * types and the stylesheet; a slice with a file or a folder that is neither its `index.ts` nor a segment;
 * a slice with no `index.ts`.
 */
export function findPluginSliceShapeViolations(files) {
  const found = []
  const entries = new Set()
  for (const { file, plugin, parts } of listShippedFrontendFiles(files)) {
    if (parts.length === 1) {
      if (!FRONTEND_ROOT_FILES.includes(parts[0])) {
        found.push({ file, line: 1, symbol: `${parts[0]} lies at the root of the frontend, which holds ${FRONTEND_ROOT_FILES.join(', ')} and the slices` })
      }
      continue
    }
    if (FRONTEND_ROOT_SEGMENTS.includes(parts[0])) continue
    if (parts.length === 2) {
      if (parts[1] === 'index.ts') entries.add(`${plugin}/${parts[0]}`)
      else found.push({ file, line: 1, symbol: `${parts[1]} lies in the slice ${parts[0]} outside its segments (${SLICE_SEGMENTS.join(', ')})` })
    } else if (!SLICE_SEGMENTS.includes(parts[1])) {
      found.push({ file, line: 1, symbol: `${parts[1]} is not a segment of the slice ${parts[0]} (${SLICE_SEGMENTS.join(', ')})` })
    }
  }
  for (const slice of listPluginSlices(files)) {
    if (!entries.has(slice)) {
      const [plugin, name] = slice.split('/')
      found.push({ file: `plugins/${plugin}/frontend/${name}`, line: 1, symbol: `the slice ${name} has no index.ts` })
    }
  }
  return found
}

/** A frontend of two slices that breaks each import rule once: a deep import into a slice from the entry, an import of a sibling slice through its index, and one through its files. */
export const PLANTED_SLICE_FRONTEND = {
  'frontend/index.ts': ["import { alpha } from './alpha'", "import { total } from './alpha/model/total'", 'export const both = [alpha, total]', ''].join('\n'),
  'frontend/alpha/index.ts': ["export { total as alpha } from './model/total'", ''].join('\n'),
  'frontend/alpha/model/total.ts': 'export const total = 1\n',
  'frontend/beta/index.ts': ["export { count } from './ui/count'", ''].join('\n'),
  'frontend/beta/ui/count.ts': ["import { alpha } from '../../alpha'", "import { total } from '../../alpha/model/total'", 'export const count = alpha + total', ''].join('\n'),
}

/** The lines of `frontend/` files that `PLANTED_SLICE_FRONTEND` must be refused at. */
export const PLANTED_SLICE_REFUSALS = { 'frontend/index.ts': [2], 'frontend/beta/ui/count.ts': [1, 2] }

/** The lines of the boundaries findings in an ESLint JSON report, by file name below the linted folder. */
export function readBoundaryLines(report, folder) {
  const lines = {}
  for (const result of report) {
    const hits = result.messages.filter((one) => one.ruleId?.startsWith('boundaries/')).map((one) => one.line)
    if (hits.length > 0) lines[path.relative(folder, result.filePath)] = [...new Set(hits)].sort((a, b) => a - b)
  }
  return lines
}

/** Lints `files` in a copy of the golden template and returns the lines of the slice findings by file. */
export function lintPlantedFrontend(files) {
  const template = path.join(REPO_ROOT, 'plugins/.example')
  const dir = fs.mkdtempSync(path.join(REPO_ROOT, 'plugins', '.slices-self-test-'))
  try {
    for (const name of ['eslint.config.js', 'numen-rules.mjs', 'tsconfig.json']) {
      fs.copyFileSync(path.join(template, name), path.join(dir, name))
    }
    for (const [name, text] of Object.entries(files)) {
      fs.mkdirSync(path.dirname(path.join(dir, name)), { recursive: true })
      fs.writeFileSync(path.join(dir, name), text)
    }
    const res = spawnSync(path.join(REPO_ROOT, 'node_modules/.bin/eslint'), ['-f', 'json', 'frontend'], { cwd: dir, encoding: 'utf8' })
    return readBoundaryLines(JSON.parse(res.stdout || '[]'), dir)
  } finally {
    fs.rmSync(dir, { recursive: true, force: true })
  }
}


/**
 * The slices of every plugin frontend hold their shape, and the rule that keeps them apart is live: the
 * slices and files it saw are counted, and a planted frontend must be refused where it breaks each import rule.
 */
export function verifyPluginFrontendSlices() {
  const files = listShippedFiles(['plugins'])
  requireMatches('plugin frontend slices', listPluginSlices(files).length)
  requireMatches('plugin frontend files', listShippedFrontendFiles(files).length)
  failOnFindings('A plugin frontend breaks its shape', findPluginSliceShapeViolations(files))
  const refused = lintPlantedFrontend(PLANTED_SLICE_FRONTEND)
  const sorted = (lines) => JSON.stringify(Object.entries(lines).sort(([a], [b]) => a.localeCompare(b)))
  if (sorted(refused) !== sorted(PLANTED_SLICE_REFUSALS)) {
    throw new Error(
      `the slice rule of plugins/.example/eslint.config.js refused ${JSON.stringify(refused)} in the planted frontend, expected ${JSON.stringify(PLANTED_SLICE_REFUSALS)}`,
    )
  }
}

export const PLUGIN_VERIFIERS = [
  {
    id: 'plugin-config-golden-parity',
    scope: 'plugins',
    module: 'plugins',
    name: 'Plugin Configs Match Golden Example plugins/.example/ & Root .golangci.yml',
    run: verifyPluginConfigGoldenParity,
  },
  {
    id: 'plugin-backend-layers',
    scope: 'plugins',
    module: 'plugins',
    name: 'Plugin Backend Internal Hexagonal Layer Direction',
    run: verifyPluginBackendInternalLayers,
  },
  {
    id: 'plugin-service-size',
    scope: 'plugins',
    module: 'plugins',
    name: 'Plugin Backend Packages Stay Within The File, Package And Service Size Limits',
    run: verifyPluginServiceSize,
  },
  {
    id: 'plugin-service-imports',
    scope: 'plugins',
    module: 'plugins',
    name: 'Plugin Backend Service Package Imports Only The Allow List',
    run: verifyPluginServiceImports,
  },
  {
    id: 'plugins-leakage',
    scope: 'plugins',
    module: 'plugins',
    name: 'Plugin Static Non-Leakage & Manifest Schema Validation',
    run: verifyPluginsStatic,
  },
  {
    id: 'plugins-use-main-entry',
    scope: 'plugins',
    module: 'plugins',
    name: 'Plugins Import the SDK and UI Through Their Main Entry (only the test entry, stylesheets and build entries a package declares in exports)',
    run: verifyPluginsUseMainEntry,
  },
  {
    id: 'plugin-frontend-slices',
    isLint: true,
    scope: 'plugins',
    module: 'plugins',
    name: 'Plugin Frontend Slices (a slice is its index and four segments; no sibling slice, no deep import; the rule is proven on a planted frontend)',
    run: verifyPluginFrontendSlices,
  },
  {
    id: 'plugins-frontend',
    isLint: true,
    scope: 'plugins',
    module: 'plugins',
    name: 'Plugin Frontend Contract & Boundaries (no @/**, no @plugins/**)',
    run: verifyPluginsFrontend,
  },
]
