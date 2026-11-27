/**
 * How the plugin mutation scores are read from the shard reports.
 */
import assert from 'node:assert/strict'
import { mkdtempSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import test from 'node:test'
import { mergeQualityFile } from '../shared/plugin-quality.mjs'
import { listMutationScores } from './quality.mjs'

const location = (line) => ({ start: { line, column: 1 }, end: { line, column: 2 } })

const writeReport = (dir, name, statuses) =>
  writeFileSync(
    join(dir, `${name}.json`),
    JSON.stringify({
      files: {
        '/repo/plugins/todos/a.go': {
          source: 'a\nb\nc\nd\n',
          mutants: statuses.map((status, index) => ({
            id: `a.go:${index}`,
            mutatorName: 'CONDITIONALS_BOUNDARY',
            replacement: `r${index}`,
            location: location(index + 1),
            status,
          })),
        },
      },
    }),
  )

const withReports = (check) => {
  const dir = mkdtempSync(join(tmpdir(), 'mutation-quality-'))
  try {
    check(dir)
  } finally {
    rmSync(dir, { recursive: true, force: true })
  }
}

test('a plugin target with every shard reported gets its score', () => {
  withReports((dir) => {
    writeReport(dir, 'plugins-todos-go-1-of-2', ['Killed', 'Survived'])
    writeReport(dir, 'plugins-todos-go-2-of-2', ['Killed', 'Timeout'])
    const targets = [{ name: 'plugins-todos-go', kind: 'go', at: 'plugins/todos', shards: 2 }]

    assert.deepEqual(listMutationScores(targets, dir, '2026-11-26T00:00:00Z'), [
      { target: 'plugins-todos-go', mutationScore: 75, measuredAt: '2026-11-26T00:00:00Z' },
    ])
  })
})

test('a plugin target with a missing shard report is left out', () => {
  withReports((dir) => {
    writeReport(dir, 'plugins-todos-1-of-2', ['Killed'])
    const targets = [{ name: 'plugins-todos', kind: 'go', at: 'plugins/todos', shards: 2 }]

    assert.deepEqual(listMutationScores(targets, dir, '2026-11-26T00:00:00Z'), [])
  })
})

test('a target outside the plugins folder is left out', () => {
  withReports((dir) => {
    writeReport(dir, 'libs-core-kernel', ['Killed'])
    const targets = [{ name: 'libs-core-kernel', kind: 'go', at: 'libs/core/kernel', shards: 1 }]

    assert.deepEqual(listMutationScores(targets, dir, '2026-11-26T00:00:00Z'), [])
  })
})

test('a stored target not measured again keeps its entry', () => {
  const previous = {
    targets: [
      { target: 'plugins-a', mutationScore: 50, measuredAt: '2026-11-01T00:00:00Z' },
      { target: 'plugins-b', mutationScore: 60, measuredAt: '2026-11-01T00:00:00Z' },
    ],
  }
  const merged = mergeQualityFile(
    previous,
    [{ target: 'plugins-b', mutationScore: 70, measuredAt: '2026-11-26T00:00:00Z' }],
    '2026-11-26T00:00:00Z',
  )

  assert.deepEqual(
    merged.targets.map((one) => [one.target, one.mutationScore]),
    [
      ['plugins-a', 50],
      ['plugins-b', 70],
    ],
  )
})
