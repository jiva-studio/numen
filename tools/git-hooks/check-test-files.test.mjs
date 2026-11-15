import test from 'node:test'
import assert from 'node:assert/strict'
import { spawnSync, execFileSync } from 'node:child_process'
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import {
  findProductionChanges,
  getTitleType,
  isTestMaterial,
  isVersionOnlyChange,
  removeDirectives,
} from './check-test-files.mjs'

test('getTitleType reads the type of a title with or without a scope', () => {
  assert.equal(getTitleType('test(sdk-go): kill mutation survivors'), 'test')
  assert.equal(getTitleType('test: cover the parser'), 'test')
  assert.equal(getTitleType('test(core,ui)!: drop a helper'), 'test')
  assert.equal(getTitleType('fix(core): read a title'), 'fix')
  assert.equal(getTitleType('testing the title'), null)
})

test('isTestMaterial accepts tests, test support and mutation config', () => {
  for (const file of [
    'libs/note/domain/match_test.go',
    'apps/desktop/editor/src/entities/note/model/buffer/notes.test.ts',
    'tests/e2e/desktop/tests/open.spec.ts',
    'libs/ui/components/src/button/Button.stories.ts',
    'libs/core/protocol/testdata/templates.json',
    'tests/e2e/desktop/fixtures/vault.ts',
    'apps/desktop/editor/src/testing/mocks/note.ts',
    'libs/core/kernel/internal/testsupport/plugintest/host.go',
    'libs/plugin/sdk-go/server/servertest/server.go',
    'libs/plugin/sdk-ts/test/setup.ts',
    'features/editor/notes/open-a-note.feature',
    'tests/acceptance/desktop/steps/notes.ts',
    'tools/ci-mutation/stryker.config.mjs',
  ]) {
    assert.equal(isTestMaterial(file), true, file)
  }
})

test('isTestMaterial refuses production code', () => {
  for (const file of [
    'libs/plugin/sdk-go/connection/client.go',
    'plugins/views-flashcards-studio/backend/domain/format/deck.go',
    'plugins/views-flashcards-studio/manifest.json',
    'apps/desktop/editor/src/entities/note/model/buffer/notes.ts',
    'plugins/connections-llm/backend/fakeopenai/server.go',
    'tools/ci-mutation/mutate.mjs',
  ]) {
    assert.equal(isTestMaterial(file), false, file)
  }
})

test('removeDirectives takes out Stryker and gomutants comments only', () => {
  const before = 'if a > b {\n\treturn a\n}'
  const after = [
    '// gomutants:disable-next-line BRANCH_IF reason="the branch returns where the function ends"',
    'if a > b { // gomutants:disable CONDITIONALS_BOUNDARY reason="equal values return a"',
    '\treturn a',
    '}',
  ].join('\n')
  assert.equal(removeDirectives(after), before)
  assert.equal(
    removeDirectives("  // Stryker disable next-line StringLiteral: interface copy\nconst a = 'x'"),
    "const a = 'x'",
  )
  assert.equal(removeDirectives('// Stryker restore all\nx()'), 'x()')
  assert.equal(removeDirectives('// a plain comment\nx()'), '// a plain comment\nx()')
})

test('isVersionOnlyChange accepts a change of "version" alone', () => {
  const manifest = (version, name = 'a') => JSON.stringify({ id: 'a', name, version }, null, 2)
  assert.equal(isVersionOnlyChange(manifest('2026.11.1'), manifest('2026.11.2')), true)
  assert.equal(isVersionOnlyChange(manifest('2026.11.1'), manifest('2026.11.2', 'b')), false)
  assert.equal(isVersionOnlyChange('not json', manifest('2026.11.2')), false)
})

test('findProductionChanges lists production files whose code changed', () => {
  const base = {
    'a/code.go': 'if a > b {\n\treturn a\n}',
    'a/other.go': 'return 1',
    'a/manifest.json': '{"id":"a","version":"2026.11.1"}',
    'b/manifest.json': '{"id":"b","version":"2026.11.1"}',
  }
  const head = {
    'a/code.go': '// gomutants:disable-next-line BRANCH_IF reason="same end"\nif a > b {\n\treturn a\n}',
    'a/other.go': 'return 2',
    'a/manifest.json': '{"id":"a","version":"2026.11.2"}',
    'b/manifest.json': '{"id":"b","version":"2026.11.2","main":"x"}',
  }
  const changes = [
    { status: 'M', file: 'a/code_test.go' },
    { status: 'M', file: 'a/code.go' },
    { status: 'M', file: 'a/other.go' },
    { status: 'M', file: 'a/manifest.json' },
    { status: 'M', file: 'b/manifest.json' },
    { status: 'A', file: 'a/new.go' },
    { status: 'D', file: 'a/old.go' },
  ]
  assert.deepEqual(
    findProductionChanges({ changes, readBase: (f) => base[f], readHead: (f) => head[f] }),
    ['a/other.go', 'b/manifest.json', 'a/new.go', 'a/old.go'],
  )
})

test('the command fails a test pull request that changes production code', (t) => {
  const dir = mkdtempSync(path.join(tmpdir(), 'check-test-files-'))
  t.after(() => rmSync(dir, { recursive: true, force: true }))
  const git = (...args) => execFileSync('git', args, { cwd: dir, encoding: 'utf8' })
  const write = (file, text) => {
    mkdirSync(path.dirname(path.join(dir, file)), { recursive: true })
    writeFileSync(path.join(dir, file), text)
  }
  git('init', '-q', '-b', 'main')
  git('config', 'user.email', 'test@example.test')
  git('config', 'user.name', 'test')
  write('lib/code.go', 'package lib\n\nfunc Max(a, b int) int {\n\tif a > b {\n\t\treturn a\n\t}\n\treturn b\n}\n')
  git('add', '.')
  git('commit', '-q', '-m', 'base')
  write('lib/code_test.go', 'package lib\n')
  git('add', '.')
  git('commit', '-q', '-m', 'tests only')

  const script = path.join(path.dirname(fileURLToPath(import.meta.url)), 'check-test-files.mjs')
  const run = (title) =>
    spawnSync('node', [script], { cwd: dir, encoding: 'utf8', env: { ...process.env, PR_TITLE: title } })

  assert.equal(run('test(lib): cover Max').status, 0)

  write('lib/code.go', 'package lib\n\nfunc Max(a, b int) int {\n\tif a >= b {\n\t\treturn a\n\t}\n\treturn b\n}\n')
  git('commit', '-q', '-am', 'production change')
  const failed = run('test(lib): cover Max')
  assert.equal(failed.status, 1)
  assert.match(failed.stderr, /lib\/code\.go/)
  assert.equal(run('fix(lib): return a on a tie').status, 0)
})
