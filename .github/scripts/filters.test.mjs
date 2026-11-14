// The filters of .github/filters.yml, matched as dorny/paths-filter matches them: js-yaml reads the
// file, a nested list is flattened, picomatch compiles each glob with `dot: true`, and
// `some-with-excludes` makes a filter true for a file that matches one glob and no `!` glob.
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { describe, it } from 'node:test'
import { load } from 'js-yaml'
import picomatch from 'picomatch'

const filters = Object.fromEntries(
  Object.entries(load(readFileSync(new URL('../filters.yml', import.meta.url), 'utf8'))).map(
    ([name, globs]) => {
      const matchers = [globs].flat(Infinity).map((glob) => picomatch(glob, { dot: true }, true))
      const includes = matchers.filter((matcher) => !matcher.state.negated)
      const excludes = matchers.filter((matcher) => matcher.state.negated)
      const isMatch = (file) =>
        includes.some((matcher) => matcher(file)) && !excludes.some((matcher) => !matcher(file))
      return [name, isMatch]
    },
  ),
)

/** The filters ci.yml and codeql.yml read. */
const AREAS = [
  'core',
  'structure',
  'protocol',
  'sdk_go',
  'desktop',
  'mobile',
  'ui',
  'desktop_fe',
  'mobile_fe',
  'plugins_all',
  'plugin_files',
  'plugin_stories',
  'plugin_versions',
  'docs',
  'references',
  'code',
  'narrative',
  'go',
  'js',
  'workflows',
  'staging',
  'mutation_tool',
]

/** The areas the files make true. */
const getAreas = (...files) => AREAS.filter((area) => files.some((file) => filters[area](file)))

describe('the filters of the changed areas', () => {
  it('defines every area the workflows read', () => {
    assert.deepEqual(
      AREAS.filter((area) => !filters[area]),
      [],
    )
  })

  it('runs every area except the plugin list for a change to the CI router', () => {
    const all = AREAS.filter((area) => area !== 'plugin_files')
    assert.deepEqual(getAreas('Makefile'), all)
    assert.deepEqual(getAreas('.github/filters.yml'), all)
    assert.deepEqual(getAreas('.github/workflows/ci.yml'), all)
  })

  it('scopes workflow, action, lock file and config changes to the areas that use them', () => {
    assert.deepEqual(getAreas('.github/workflows/go.yml'), [
      'core',
      'structure',
      'protocol',
      'sdk_go',
      'desktop',
      'mobile',
      'workflows',
    ])
    assert.deepEqual(getAreas('.github/actions/build-ui/action.yml'), [
      'ui',
      'desktop_fe',
      'mobile_fe',
      'plugins_all',
      'plugin_stories',
      'workflows',
    ])
    assert.deepEqual(getAreas('go.work'), [
      'core',
      'structure',
      'protocol',
      'sdk_go',
      'desktop',
      'mobile',
      'plugins_all',
      'code',
      'go',
    ])
    assert.deepEqual(getAreas('tsconfig.base.json'), [
      'ui',
      'desktop_fe',
      'mobile_fe',
      'plugins_all',
      'plugin_stories',
      'docs',
      'code',
      'js',
    ])
    const lockAreas = getAreas('package-lock.json')
    for (const area of ['ui', 'desktop_fe', 'mobile_fe', 'plugins_all', 'plugin_stories', 'core', 'protocol']) {
      assert.ok(lockAreas.includes(area), area)
    }
    for (const area of ['sdk_go', 'desktop', 'mobile', 'go', 'staging', 'mutation_tool']) {
      assert.ok(!lockAreas.includes(area), area)
    }
  })

  it('runs only the narrative check for documentation', () => {
    assert.deepEqual(getAreas('docs/plan.md', 'libs/core/kernel/README.md'), ['narrative'])
  })

  it('lists a plugin for a change inside it and runs no Go module', () => {
    const areas = getAreas('plugins/readers-pdf/frontend/View.vue')
    for (const area of ['plugin_files', 'plugin_stories', 'references', 'docs', 'js', 'code'])
      assert.ok(areas.includes(area), area)
    for (const area of ['plugins_all', 'core', 'desktop', 'go']) assert.ok(!areas.includes(area), area)
    assert.ok(!getAreas('plugins/engines-transcription/backend/main.go').includes('plugin_stories'))
  })

  it('runs the Go structure checks and not the core for a change to a plugin backend', () => {
    const areas = getAreas('plugins/engines-transcription/backend/main.go')
    for (const area of ['structure', 'plugin_files', 'go']) assert.ok(areas.includes(area), area)
    for (const area of ['core', 'plugins_all']) assert.ok(!areas.includes(area), area)
    assert.ok(!getAreas('plugins/engines-transcription/backend/README.md').includes('structure'))
  })

  it('runs the core and the manual for a changed manifest', () => {
    const areas = getAreas('plugins/views-file-tree/manifest.json')
    assert.ok(areas.includes('core'))
    assert.ok(areas.includes('docs'))
  })

  it('runs every plugin for a change to the golden plugin template', () => {
    const areas = getAreas('plugins/.example/Makefile')
    assert.ok(areas.includes('plugins_all'))
    assert.ok(!areas.includes('plugin_files'))
  })

  it('runs what reads the Go SDK and not the window frontend', () => {
    const areas = getAreas('libs/plugin/sdk-go/host/store.go')
    assert.ok(areas.includes('sdk_go'))
    assert.ok(areas.includes('plugins_all'))
    assert.ok(!areas.includes('desktop_fe'))
    assert.ok(!areas.includes('plugin_stories'))
  })

  it('runs the Go modules that build on the core and no frontend for a change to the core', () => {
    assert.deepEqual(getAreas('libs/core/kernel/usecase/search/search.go'), [
      'core',
      'structure',
      'desktop',
      'mobile',
      'code',
      'narrative',
      'go',
    ])
  })

  it('runs every Go module, every frontend and every plugin for a change to the protocol', () => {
    const areas = getAreas('libs/core/protocol/numen/v1/manifest.proto')
    for (const area of ['protocol', 'core', 'desktop', 'mobile', 'sdk_go', 'desktop_fe', 'mobile_fe', 'plugins_all']) {
      assert.ok(areas.includes(area), area)
    }
    assert.ok(!areas.includes('ui'))
  })

  it('runs the frontends that build on the components', () => {
    const areas = getAreas('libs/ui/components/src/Button.vue')
    for (const area of ['ui', 'desktop_fe', 'mobile_fe', 'references', 'plugin_stories', 'plugins_all']) {
      assert.ok(areas.includes(area), area)
    }
    assert.ok(!areas.includes('core'))
  })

  it('runs the window frontend and the manual for a change to the editor', () => {
    const areas = getAreas('apps/desktop/editor/src/app/App.vue')
    for (const area of ['desktop_fe', 'docs', 'references']) assert.ok(areas.includes(area), area)
    for (const area of ['desktop', 'ui', 'plugins_all']) assert.ok(!areas.includes(area), area)
  })

  it('lints the end-to-end package for a change to its tests', () => {
    assert.ok(getAreas('tests/e2e/desktop/tests/plugins/system/startup.spec.ts').includes('desktop_fe'))
    assert.ok(getAreas('tests/e2e/cli/plugin-commands.sh').includes('desktop'))
  })

  it('splits the mobile module into its Go binding and its screens', () => {
    assert.deepEqual(getAreas('apps/mobile/bind/bind.go'), ['mobile', 'code', 'narrative', 'go'])
    assert.ok(getAreas('apps/mobile/src/App.vue').includes('mobile_fe'))
    assert.ok(!getAreas('apps/mobile/src/App.vue').includes('mobile'))
  })

  it('runs the workflow checks alone for a workflow that is not CI', () => {
    assert.deepEqual(getAreas('.github/workflows/release.yml'), ['workflows'])
  })

  it('runs the mutation tool tests for a change to the tool', () => {
    assert.ok(getAreas('tools/ci-mutation/shards.mjs').includes('mutation_tool'))
  })

  it('leaves the landing page to its own workflow', () => {
    assert.deepEqual(getAreas('apps/landing/src/pages/index.astro'), [])
  })
})
