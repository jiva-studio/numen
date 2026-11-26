/**
 * Which modules coverage measures, what the per-plugin quality file holds,
 * and the overall badge.
 */
import assert from 'node:assert/strict'
import { mkdtempSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import test from 'node:test'
import * as collect from './collect.mjs'
import { getTargets } from '../ci-mutation/targets.mjs'

const findModule = (at, kind) =>
  collect.listCoverageModules().find((one) => one.at === at && one.kind === kind)

/* -------------------------------- Discovery ------------------------------- */

test('every module and package the mutation discovery finds is measured, and no other', () => {
  const measured = collect
    .listCoverageModules()
    .map(({ name, kind, at }) => ({ name, kind, at }))
    .sort((a, b) => a.name.localeCompare(b.name))
  const targets = getTargets()
    .map(({ name, kind, at }) => ({ name, kind, at }))
    .sort((a, b) => a.name.localeCompare(b.name))
  assert.deepEqual(measured, targets)
})

test('Go modules with tests of their own are measured, the ones the hand-written list missed too', () => {
  for (const at of [
    'libs/core/kernel',
    'libs/plugin/sdk-go',
    'libs/plugin/catalog',
    'libs/shared/note',
    'plugins/views-flashcards-studio',
    'plugins/views-media-viewer',
    'plugins/agents-mcp-server',
  ]) {
    assert.ok(findModule(at, 'go'), at)
  }
})

test('a Go module without a test of its own is not measured', () => {
  const folders = collect.listCoverageModules().map((one) => one.at)
  assert.ok(!folders.includes('apps/desktop/build/pgonorm'))
  assert.ok(!folders.includes('apps/desktop/build/tools'))
})

test('TypeScript packages and plugin frontends are measured', () => {
  for (const at of [
    'libs/ui/components',
    'apps/desktop/editor',
    'apps/desktop/flashcards',
    'plugins/views-plex-graph',
    'plugins/notes',
    'plugins/todos',
  ]) {
    assert.ok(findModule(at, 'ts'), at)
  }
})

/* ----------------------------- Plugin quality ----------------------------- */

const measureAll = (total, covered) =>
  collect.listCoverageModules().map((one) => ({ ...one, total, covered }))

test('the quality file holds the coverage of a Go plugin and of a frontend-only plugin', () => {
  const measuredAt = '2026-11-26T03:00:00.000Z'
  const entries = collect.listPluginCoverage(measureAll(200, 150), measuredAt)
  assert.deepEqual(entries.find((one) => one.target === 'plugins-agents-mcp-server'), {
    target: 'plugins-agents-mcp-server',
    coverage: 75,
    measuredAt,
  })
  assert.deepEqual(entries.find((one) => one.target === 'plugins-views-plex-graph'), {
    target: 'plugins-views-plex-graph',
    coverage: 75,
    measuredAt,
  })
})

test('a plugin with Go and a frontend has a coverage entry for each, named by kind', () => {
  const entries = collect.listPluginCoverage(measureAll(3, 2), '2026-11-26T03:00:00.000Z')
  const todos = entries.filter((one) => one.target.startsWith('plugins-todos'))
  assert.deepEqual(
    todos.map((one) => [one.target, one.coverage]).sort(),
    [
      ['plugins-todos-go', 66.7],
      ['plugins-todos-ts', 66.7],
    ],
  )
})

test('the quality file holds plugins only, and none without statements', () => {
  const measured = [
    { name: 'libs-core-kernel', kind: 'go', at: 'libs/core/kernel', total: 10, covered: 5 },
    { name: 'libs-ui-components', kind: 'ts', at: 'libs/ui/components', total: 10, covered: 5 },
    { name: 'plugins-notes', kind: 'ts', at: 'plugins/notes', total: 0, covered: 0 },
    { name: 'plugins-tools-logs', kind: 'ts', at: 'plugins/tools-logs', total: 4, covered: 1 },
  ]
  assert.deepEqual(
    collect.listPluginCoverage(measured, 'now').map((one) => one.target),
    ['plugins-tools-logs'],
  )
})

/* ---------------------------------- Badge --------------------------------- */

test('a Go coverprofile counts its statements and the covered ones', () => {
  const dir = mkdtempSync(join(tmpdir(), 'coverage-'))
  try {
    const profile = join(dir, 'kernel.out')
    writeFileSync(
      profile,
      [
        'mode: set',
        'github.com/jiva-studio/numen/libs/core/kernel/note.go:10.2,12.3 3 1',
        'github.com/jiva-studio/numen/libs/core/kernel/note.go:14.2,15.3 2 0',
        'github.com/jiva-studio/numen/libs/core/kernel/link.go:4.2,9.3 5 4',
        '',
      ].join('\n'),
    )
    assert.deepEqual(collect.parseGoProfile(profile), { total: 10, covered: 8 })
    assert.deepEqual(collect.parseGoProfile(join(dir, 'missing.out')), { total: 0, covered: 0 })
  } finally {
    rmSync(dir, { recursive: true, force: true })
  }
})

test('the badge weighs Go and TypeScript by their statements', () => {
  const measured = [
    { name: 'libs-core-kernel', kind: 'go', total: 300, covered: 270 },
    { name: 'libs-ui-components', kind: 'ts', total: 100, covered: 70 },
  ]
  assert.deepEqual(collect.getCoverageBadge(measured), {
    schemaVersion: 1,
    label: 'coverage',
    message: '85%',
    color: 'brightgreen',
  })
})

test('the badge turns yellow below 85% and red below 70%', () => {
  const badge = (covered) =>
    collect.getCoverageBadge([{ name: 'libs-core-kernel', kind: 'go', total: 100, covered }])
  assert.equal(badge(84).color, 'yellow')
  assert.equal(badge(70).color, 'yellow')
  assert.equal(badge(69).color, 'red')
  assert.equal(badge(69).message, '69%')
})

test('the badge of nothing measured reads 0%', () => {
  assert.equal(collect.getCoverageBadge([]).message, '0%')
})
