import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import { AREAS, getAreas } from './changes.mjs'

const plugins = ['readers-pdf', 'views-file-tree', 'themes-presets']
const e2eFolders = ['readers.pdf', 'system']
const getChange = (...files) => getAreas({ everything: false, files, plugins, e2eFolders })
const getTrue = (output) => AREAS.filter((area) => output[area] === 'true')

describe('the areas of a change', () => {
  it('runs every area and every plugin on a push', () => {
    const output = getAreas({ everything: true, files: [], plugins, e2eFolders })
    assert.deepEqual(getTrue(output), AREAS)
    assert.equal(output.plugin_dirs, 'plugins/readers-pdf plugins/themes-presets plugins/views-file-tree')
    assert.equal(output.e2e, 'all')
  })

  it('runs every area for a change to the root lock file or to CI itself', () => {
    assert.deepEqual(getTrue(getChange('package-lock.json')), AREAS)
    assert.deepEqual(getTrue(getChange('.github/actions/build-ui/action.yml')), AREAS)
  })

  it('runs only the narrative check for documentation', () => {
    const output = getChange('docs/plan.md', 'libs/core/README.md')
    assert.deepEqual(getTrue(output), ['narrative'])
    assert.equal(output.plugins, 'false')
    assert.equal(output.e2e, 'none')
  })

  it('runs one plugin and its end-to-end folder for a change inside it', () => {
    const output = getChange('plugins/readers-pdf/frontend/View.vue')
    assert.equal(output.plugin_dirs, 'plugins/readers-pdf')
    assert.equal(output.e2e, 'readers.pdf')
    assert.equal(output.core, 'false')
    assert.equal(output.desktop, 'false')
    assert.equal(output.references, 'true')
    assert.equal(output.js, 'true')
    assert.equal(output.go, 'false')
  })

  it('runs no end-to-end folder for a plugin the suite has none for', () => {
    assert.equal(getChange('plugins/views-file-tree/backend/main.go').e2e, 'none')
  })

  it('runs the core and the manual for a changed manifest', () => {
    const output = getChange('plugins/views-file-tree/manifest.json')
    assert.equal(output.core, 'true')
    assert.equal(output.docs, 'true')
  })

  it('runs every plugin for a change to the golden plugin template', () => {
    assert.equal(getChange('plugins/.example/Makefile').plugin_dirs.split(' ').length, 3)
  })

  it('runs what reads the Go SDK and not the window frontend', () => {
    const output = getChange('libs/plugin-sdk-go/host/store.go')
    assert.equal(output.sdk_go, 'true')
    assert.equal(output.plugins, 'true')
    assert.equal(output.desktop_fe, 'false')
    assert.equal(output.e2e, 'all')
  })

  it('runs the Go modules that build on the core and no frontend for a change to the core', () => {
    const output = getChange('libs/core/usecase/search/search.go')
    assert.deepEqual(getTrue(output), ['core', 'desktop', 'mobile', 'code', 'narrative', 'go'])
  })

  it('runs the frontends that build on the components', () => {
    const output = getChange('libs/ui/core/src/Button.vue')
    for (const area of ['ui', 'desktop_fe', 'mobile_fe', 'references', 'plugin_stories']) {
      assert.equal(output[area], 'true', area)
    }
    assert.equal(output.core, 'false')
  })

  it('runs one end-to-end folder for a change to its tests', () => {
    const output = getChange('tests/e2e/desktop/tests/plugins/system/lifecycle.spec.ts')
    assert.equal(output.e2e, 'system')
    assert.equal(output.desktop_fe, 'true')
  })

  it('splits the mobile module into its Go binding and its screens', () => {
    assert.deepEqual(getTrue(getChange('apps/mobile/bind/bind.go')), ['mobile', 'code', 'narrative', 'go'])
    assert.equal(getChange('apps/mobile/src/App.vue').mobile_fe, 'true')
    assert.equal(getChange('apps/mobile/src/App.vue').mobile, 'false')
  })

  it('runs the workflow checks alone for a workflow that is not CI', () => {
    assert.deepEqual(getTrue(getChange('.github/workflows/release.yml')), ['workflows'])
  })

  it('runs the mutation tool tests for a change to the tool', () => {
    assert.equal(getChange('tools/ci-mutation/shards.mjs').mutation_tool, 'true')
  })

  it('leaves the landing page to its own workflow', () => {
    assert.deepEqual(getTrue(getChange('apps/landing/src/pages/index.astro')), [])
  })
})
