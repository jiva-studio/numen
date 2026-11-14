import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import { test } from 'node:test'
import { fileURLToPath } from 'node:url'
import { getPlan, getPluginTokens, getScope, getScopePaths, planJobs } from './select-tests.mjs'

const e2eRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const pluginsRoot = path.resolve(e2eRoot, '../../../plugins')

await test('a plugin change selects its folder, the specs that name it and the acceptance scenarios', () => {
  assert.equal(
    getScope(['plugins/readers-pdf/frontend/reader.ts']),
    'readers.pdf acceptance',
  )
})

await test('a plugin without tests of its own selects the shell alone', () => {
  assert.equal(getScope(['plugins/agents-mcp-server/backend/main.go']), 'shell/')
  assert.deepEqual(getScopePaths('shell/'), ['tests/shell/'])
})

await test('a change to a spec folder selects that folder', () => {
  assert.equal(getScope(['tests/e2e/desktop/tests/plugins/notes/wikilinks.spec.ts']), 'notes')
  assert.equal(
    getScope([
      'tests/e2e/desktop/tests/plugins/system/settings.spec.ts',
      'plugins/tools-updater/backend/main.go',
    ]),
    'system acceptance',
  )
})

await test('a change to the window, the support code or the lock file selects every test', () => {
  for (const file of [
    'libs/core/kernel/usecase/note/create.go',
    'apps/desktop/editor/src/main.ts',
    'tests/e2e/desktop/tests/support/workspace.ts',
    'tests/e2e/desktop/playwright.config.ts',
    'tests/vault/notes/Entropy.md',
    'package-lock.json',
    'Makefile',
    'go.work',
    '.github/workflows/desktop-e2e.yml',
    '.github/actions/setup-playwright/action.yml',
  ]) {
    assert.equal(getScope(['plugins/notes/frontend/index.ts', file]), 'all')
  }
})

await test('a change outside the desktop window, or to its documentation, selects none', () => {
  assert.equal(
    getScope([
      'apps/mobile/src/main.ts',
      'docs/index.md',
      'plugins/notes/README.md',
      'libs/core/kernel/usecase/note/create_test.go',
      'apps/desktop/editor/src/main.test.ts',
      'libs/ui/components/src/Button.stories.ts',
      'plugins/notes/frontend/Note.test.ts',
      'plugins/.golangci.yml',
      'apps/desktop/editor/eslint.config.js',
      'apps/desktop/editor/vitest.config.ts',
      'libs/ui/components/.storybook/main.ts',
      'tests/e2e/desktop/scripts/select-tests.test.mjs',
    ]),
    'none',
  )
  assert.equal(getScope(['tests/e2e/desktop/scripts/select-tests.mjs']), 'all')
  assert.deepEqual(getPlan('none'), [
    'desktop=false',
    'paths=',
    'flashcards=false',
    'acceptance=false',
    'matrix={"include":[]}',
  ])
})

await test('a change to a scenario or its steps selects the acceptance scenarios alone', () => {
  assert.equal(getScope(['features/notes/changes-on-disk.feature']), 'acceptance')
  assert.equal(getScope(['tests/acceptance/desktop/drivers/person.ts']), 'acceptance')
  assert.deepEqual(getPlan('acceptance'), [
    'desktop=false',
    'paths=',
    'flashcards=false',
    'acceptance=true',
    'matrix={"include":[]}',
  ])
  assert.equal(
    getScope(['features/notes/changes-on-disk.feature', 'tests/e2e/desktop/tests/plugins/notes/wikilinks.spec.ts']),
    'notes acceptance',
  )
  assert.deepEqual(getScopePaths('notes acceptance'), ['tests/plugins/notes/', 'tests/shell/'])
  assert.ok(getPlan('all').includes('acceptance=true'))
  assert.ok(getPlan('notes').includes('acceptance=false'))
})

await test('a change to the flashcards plugin or its tests starts the flashcards job', () => {
  const scope = getScope(['plugins/views-flashcards-studio/frontend/index.ts'])
  assert.equal(scope, 'flashcards acceptance')
  assert.ok(getPlan(scope).includes('flashcards=true'))
  assert.equal(getScope(['tests/e2e/desktop/flashcards/review.spec.ts']), 'flashcards')
})

await test('a scope of folders runs them and the shell', () => {
  assert.deepEqual(getScopePaths('readers.epub notes'), [
    'tests/plugins/notes/',
    'tests/plugins/readers.epub/',
    'tests/shell/',
  ])
  assert.ok(getPlan('notes').includes('flashcards=false'))
  assert.ok(getPlan('flashcards').includes('flashcards=true'))
  assert.deepEqual(getScopePaths('flashcards'), ['tests/shell/'])
})

await test('every plugin maps to tests that exist, and every plugin test folder has a plugin', () => {
  const reached = new Set()
  for (const dir of fs.readdirSync(pluginsRoot)) {
    if (!fs.existsSync(path.join(pluginsRoot, dir, 'manifest.json'))) continue
    for (const one of getScopePaths(getScope([`plugins/${dir}/a.ts`]))) {
      assert.ok(fs.existsSync(path.join(e2eRoot, one)), `${dir}: ${one} does not exist`)
      reached.add(one)
    }
    assert.ok(getPluginTokens(dir))
  }
  // tests/plugins/system tests the plugin host, which a change to libs/ or apps/desktop/ runs.
  const unreached = fs
    .readdirSync(path.join(e2eRoot, 'tests/plugins'))
    .filter((one) => one !== 'system' && !reached.has(`tests/plugins/${one}/`))
  assert.deepEqual(unreached, [])
})

await test('every test runs in at most four shards an engine', () => {
  const jobs = planJobs(undefined)
  assert.ok(jobs.length >= 2 && jobs.length <= 8)
  assert.deepEqual(jobs[0], { engine: 'chromium', shard: 1, shards: jobs.length / 2 })
})

await test('a change to the connections plugin selects the shell and the acceptance scenarios', () => {
  assert.equal(getScope(['plugins/connections-llm/backend/fakeopenai/server.go']), 'shell/ acceptance')
})
