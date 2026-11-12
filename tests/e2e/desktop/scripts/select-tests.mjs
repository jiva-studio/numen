// Chooses the end-to-end tests a change needs, and the CI jobs that run them.
//
//   git diff --name-only base...head | node scripts/select-tests.mjs scope
//   node scripts/select-tests.mjs plan <scope>
//
// A scope is `all`, `none`, or tokens separated by spaces. A token without `/` is a folder under
// tests/plugins/ (`readers.epub`); a token with `/` is a folder (ending in `/`) or a spec file
// under tests/ (`scenarios/link.spec.ts`). A scope of tokens runs them and tests/shell/.
//
// `scope` reads changed paths, relative to the repository root, and prints `scope=<scope>`. A
// change to plugins/<dir>/ selects the tests of that plugin: tests/plugins/<id without md.numen.>/,
// the spec files that name its id, and the files PLUGIN_TESTS adds. A change to a spec folder or
// file under tests/e2e/desktop/tests/ selects it. A change elsewhere in what the desktop window
// is built from selects every test; a change outside it, to documentation, or to a unit test,
// story or linter config outside the e2e suite selects none.
//
// `plan` prints `key=value` lines for $GITHUB_OUTPUT:
//
//   desktop=true|false     whether any test of the desktop window runs
//   paths=<paths>          the folders and spec files to run, separated by spaces; empty for all
//   flashcards=true|false  whether the flashcards window's tests run
//   matrix=<json>          {"include":[{"engine","shard","shards"}]}: one job per engine and shard
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { listSpecs, readSelectedDurations } from './shards.mjs'

const e2eRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const repoRoot = path.resolve(e2eRoot, '../../..')

const ENGINES = ['chromium', 'webkit']
/** The seconds of tests, by shards.json, one shard aims at. */
const SHARD_SECONDS = 300
const MAX_SHARDS = 4
const ID_PREFIX = 'md.numen.'
const FLASHCARDS = 'views.flashcards.studio'

/** Tests of a plugin beyond its own folder and the spec files that name its id, as scope tokens. */
export const PLUGIN_TESTS = {
  'agents-numen': ['agents.chat'],
  'agents-tools-vault': ['agents.chat'],
  'agents-tools-window': ['agents.chat'],
  'views-agent-chat': ['agents.chat'],
  'connections-llm': [
    'plugins/engines.embeddings/connection.spec.ts',
    'scenarios/connections-proofreading.spec.ts',
  ],
  'engines-ocr': [
    'scenarios/ocr-disabled.spec.ts',
    'scenarios/proofreading.spec.ts',
    'scenarios/recognition.spec.ts',
  ],
  'engines-transcription': ['scenarios/proofreading.spec.ts', 'scenarios/recording.spec.ts'],
  'readers-pdf': ['scenarios/ocr-disabled.spec.ts', 'scenarios/recognition.spec.ts'],
  'tools-media-importer': ['scenarios/link.spec.ts'],
  'tools-proofreading': [
    'scenarios/connections-proofreading.spec.ts',
    'scenarios/proofreading.spec.ts',
  ],
  'views-media-viewer': ['scenarios/recording.spec.ts'],
}

/** Paths the desktop window and its tests are built from; a change elsewhere runs no test. */
const DESKTOP =
  /^(apps\/desktop\/|libs\/|plugins\/|tests\/e2e\/desktop\/|tests\/vault\/|tools\/(build-plugin-index|shared)\/|\.github\/(workflows\/desktop-e2e\.yml$|actions\/(setup-playwright|setup-webkit|build-ui)\/)|(Makefile|package\.json|package-lock\.json|go\.work(\.sum)?|\.node-version)$)/

/** Unit tests, stories and linter or unit-test configs outside the e2e suite. */
const NON_E2E_TEST_OR_CONFIG =
  /(_test\.go|\.(test|spec|stories)\.[cm]?[jt]sx?|(^|\/)(\.golangci[^/]*\.ya?ml|eslint\.config\.[cm]?js|vitest\.config\.ts|\.storybook\/.*))$/

/** Text that no build or test reads. The vault of the tests is notes, and is read. */
const isDocumentation = (file) => /\.mdx?$/.test(file) && !file.startsWith('tests/vault/')

/** Unit tests, stories and linter configs that neither the binaries nor the e2e suite run. */
const isUnitTestOrStory = (file) =>
  file === 'tests/e2e/desktop/scripts/select-tests.test.mjs' ||
  (!file.startsWith('tests/e2e/desktop/') && NON_E2E_TEST_OR_CONFIG.test(file))

/** Whether `file` is part of what the desktop window, its plugins or its e2e suite are built from. */
const isDesktopInput = (file) =>
  DESKTOP.test(file) && !isDocumentation(file) && !isUnitTestOrStory(file)

/** The id of the plugin in plugins/<dir>, from its manifest. */
const readPluginId = (dir) =>
  JSON.parse(fs.readFileSync(path.join(repoRoot, 'plugins', dir, 'manifest.json'), 'utf8')).id

/** The spec files outside tests/shell/, which every scope runs, whose text names `id`, as scope tokens. */
const findSpecsNaming = (id) =>
  listSpecs()
    .filter((spec) => !spec.startsWith('tests/shell/'))
    .filter((spec) => {
      const text = fs.readFileSync(path.join(e2eRoot, spec), 'utf8')
      return new RegExp(`['"\`]${id.replaceAll('.', '\\.')}['"\`]`).test(text)
    })
    .map((spec) => spec.slice('tests/'.length))

/** The scope tokens of the plugin in plugins/<dir>. */
export const getPluginTokens = (dir) => {
  const id = readPluginId(dir)
  const folder = id.startsWith(ID_PREFIX) ? id.slice(ID_PREFIX.length) : id
  const own = fs.existsSync(path.join(e2eRoot, 'tests/plugins', folder)) ? [folder] : []
  return [...own, ...findSpecsNaming(id), ...(PLUGIN_TESTS[dir] ?? [])]
}

/** The path under the e2e folder that a scope token names. */
const getTokenPath = (token) => `tests/${token.includes('/') ? token : `plugins/${token}/`}`

/** The scope tokens one changed file adds, or null where the file needs every test. */
const getFileTokens = (file) => {
  const folder = /^tests\/e2e\/desktop\/tests\/plugins\/([^/]+)\//.exec(file)
  if (folder) return [folder[1]]
  const spec = /^tests\/e2e\/desktop\/tests\/(shell\/|scenarios\/.+\.spec\.ts$)/.exec(file)
  if (spec) return [spec[1]]
  if (file.startsWith('tests/e2e/desktop/flashcards/')) return [FLASHCARDS]
  const plugin = /^plugins\/([^/]+)\//.exec(file)
  if (plugin && fs.existsSync(path.join(repoRoot, 'plugins', plugin[1], 'manifest.json'))) {
    return getPluginTokens(plugin[1])
  }
  return null
}

/** The tokens in the order they are run, without the ones a folder token already holds. */
const joinTokens = (tokens) => {
  const folders = [...tokens].map(getTokenPath).filter((one) => one.endsWith('/'))
  return [...tokens]
    .filter(
      (one) =>
        !folders.some(
          (folder) => getTokenPath(one).startsWith(folder) && getTokenPath(one) !== folder,
        ),
    )
    .sort((a, b) => a.localeCompare(b))
    .join(' ')
}

/** The scope of the changed files. */
export const getScope = (files) => {
  const tokens = new Set()
  let isTested = false
  for (const file of files) {
    if (!isDesktopInput(file)) continue
    isTested = true
    const added = getFileTokens(file)
    if (added === null) return 'all'
    for (const one of added) tokens.add(one)
  }
  if (tokens.size === 0) return isTested ? 'shell/' : 'none'
  return joinTokens(tokens)
}

/**
 * The test paths of a scope: undefined for every test, an empty list for none, or the paths of its
 * tokens and tests/shell/.
 */
export const getScopePaths = (scope) => {
  const tokens = scope.split(' ').filter(Boolean)
  if (tokens.includes('all')) return undefined
  const named = tokens.filter((one) => one !== 'none')
  if (named.length === 0) return []
  return [...new Set(['tests/shell/', ...named.map(getTokenPath)])].sort((a, b) =>
    a.localeCompare(b),
  )
}

/** The jobs that run `paths` (every test when undefined): each engine split into shards of SHARD_SECONDS. */
export const planJobs = (paths) => {
  const seconds = Object.values(readSelectedDurations(paths)).reduce((sum, one) => sum + one, 0)
  if (seconds === 0) return []
  const shards = Math.min(MAX_SHARDS, Math.ceil(seconds / SHARD_SECONDS))
  return ENGINES.flatMap((engine) =>
    Array.from({ length: shards }, (_, index) => ({ engine, shard: index + 1, shards })),
  )
}

/** The `key=value` lines of the jobs a scope runs. */
export const getPlan = (scope) => {
  const paths = getScopePaths(scope)
  const jobs = paths?.length === 0 ? [] : planJobs(paths)
  const flashcards = !paths || paths.includes(getTokenPath(FLASHCARDS))
  return [
    `desktop=${jobs.length > 0}`,
    `paths=${(paths ?? []).join(' ')}`,
    `flashcards=${flashcards}`,
    `matrix=${JSON.stringify({ include: jobs })}`,
  ]
}

const isMain = process.argv[1] && fileURLToPath(import.meta.url) === path.resolve(process.argv[1])
if (isMain) {
  const [command, ...rest] = process.argv.slice(2)
  if (command === 'scope') {
    console.log(`scope=${getScope(fs.readFileSync(0, 'utf8').split('\n').filter(Boolean))}`)
  } else if (command === 'plan') {
    console.log(getPlan(rest.join(' ')).join('\n'))
  } else {
    console.error('select-tests: name `scope` or `plan <scope>`')
    process.exit(1)
  }
}
