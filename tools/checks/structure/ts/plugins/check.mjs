import { spawnSync } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'
import { parse as parseYaml } from 'yaml'
import { listPluginNames, readPluginManifests } from '../../../../shared/manifests.mjs'
import { findPluginCommandMismatches } from '../../../../shared/registrations.mjs'
import { createDeterminismSelectors } from '../base-eslint.mjs'
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
const NON_INNER_BACKEND_PACKAGES = ['cmd', ...OUTER_BACKEND_PACKAGES, 'testsupport']

/** Plugin backend internal layers (inner packages under backend/<pkg> must not import backend/service or backend/engine). */
function findPluginBackendLayerInversions() {
  const found = []
  for (const file of listShippedFiles(['plugins/*.go'])) {
    const [root, plugin, tier, subpkg] = file.split('/')
    if (root !== 'plugins' || tier !== 'backend' || !subpkg || subpkg.endsWith('.go')) continue
    if (NON_INNER_BACKEND_PACKAGES.includes(subpkg)) continue
    const text = fs.readFileSync(path.join(REPO_ROOT, file), 'utf8')
    const { imports } = getGoReferences(text)
    for (const { line, value } of imports) {
      if (
        OUTER_BACKEND_PACKAGES.some(
          (outer) => value.endsWith(`/backend/${outer}`) || value.includes(`/backend/${outer}/`),
        )
      ) {
        found.push({
          file,
          line,
          symbol: `inner package backend/${subpkg} imports outer ${value} in plugins/${plugin}`,
        })
      }
    }
  }
  return found
}

export function verifyPluginBackendInternalLayers() {
  failOnFindings(
    'Plugin backend inner package imports outer service/engine package',
    findPluginBackendLayerInversions(),
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

/** Packages that reach the machine or hold shared state; a service package does this through an adapter. */
const MACHINE_ACCESS_IMPORTS = [
  'os',
  'os/exec',
  'os/signal',
  'os/user',
  'runtime',
  'io/fs',
  'net',
  'database/sql',
  'syscall',
  'sync',
  'sync/atomic',
]

/** `net/http` names status codes and the handler type of the SDK server, so only the other packages under `net` are machine access. */
function isMachineAccessImport(value) {
  if (MACHINE_ACCESS_IMPORTS.includes(value)) return true
  if (value === 'net/http') return false
  return ['net/', 'runtime/', 'database/sql/', 'syscall/'].some((prefix) => value.startsWith(prefix))
}

/**
 * Finds the imports of a backend `service` package that reach the machine (process environment,
 * processes, runtime, file system, network, database, system calls) or hold shared state
 * (mutexes, atomics). `files` holds `{ file, imports }` with each import as `{ line, value }`.
 */
export function findServiceMachineAccess(files) {
  const found = []
  for (const { file, imports } of files) {
    for (const { line, value } of imports) {
      if (isMachineAccessImport(value)) {
        found.push({
          file,
          line,
          symbol: `service imports ${value}; machine access and state belong in internal/adapter or a use case`,
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
    'Plugin backend service package reaches the machine or holds state',
    findServiceMachineAccess(files),
  )
}

function listPluginBackends() {
  const byPlugin = new Map()
  for (const file of listShippedFiles(['plugins/*/backend/*.go', 'plugins/*/backend/**/*.go'])) {
    const [, plugin, , ...rest] = file.split('/')
    const entry = byPlugin.get(plugin) ?? { plugin, serviceLines: 0, innerPackages: new Set() }
    byPlugin.set(plugin, entry)
    if (rest.length < 2) continue
    if (rest[0] === 'service') {
      entry.hasService = true
      entry.serviceLines += fs.readFileSync(path.join(REPO_ROOT, file), 'utf8').split('\n').length - 1
    } else {
      entry.innerPackages.add(rest.slice(0, -1).join('/'))
    }
  }
  return [...byPlugin.values()]
    .filter((entry) => entry.hasService)
    .map((entry) => ({ ...entry, innerPackages: [...entry.innerPackages] }))
}

export function verifyPluginServiceSize() {
  const backends = listPluginBackends()
  requireMatches('plugin backends with a service package', backends.length)
  failOnFindings('Plugin backend service package is too large', findOversizedServices(backends))
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
  const exampleGolangci = path.join(exampleDir, '.golangci.yml')
  const baseEslint = path.join(REPO_ROOT, 'tools/lint-layers/base-eslint.mjs')
  const exampleEslint = path.join(exampleDir, 'eslint.config.js')

  if (fs.existsSync(rootGolangci) && fs.existsSync(exampleGolangci)) {
    const rootLinters = parseEnabledGoLinters(fs.readFileSync(rootGolangci, 'utf8'))
    const exLinters = parseEnabledGoLinters(fs.readFileSync(exampleGolangci, 'utf8'))
    for (const l of rootLinters) {
      if (!exLinters.includes(l)) {
        found.push({
          file: 'plugins/.example/.golangci.yml',
          line: 1,
          symbol: `missing linter '${l}' enabled in root .golangci.yml`,
        })
      }
    }
    for (const l of exLinters) {
      if (!rootLinters.includes(l)) {
        found.push({
          file: 'plugins/.example/.golangci.yml',
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
  const goConfigCopies = [
    'plugins/.example',
    'libs/plugin/sdk-go',
    ...pluginNames
      .filter((name) => fs.existsSync(path.join(pluginsDir, name, 'go.mod')))
      .map((name) => `plugins/${name}`),
  ]
  for (const name of ['.golangci.yml', '.golangci-structure.yml']) {
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
    name: 'Plugin Backend Service Package Stays The Composition Root And Wire',
    run: verifyPluginServiceSize,
  },
  {
    id: 'plugin-service-imports',
    scope: 'plugins',
    module: 'plugins',
    name: 'Plugin Backend Service Package Imports No Machine Access Or Shared State',
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
