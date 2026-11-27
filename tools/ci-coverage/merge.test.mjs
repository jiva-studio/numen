/**
 * Measuring one kind with `--kind` and merging both kinds with `--merge`.
 */
import assert from 'node:assert/strict'
import { mkdtempSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import test from 'node:test'
import * as collect from './collect.mjs'

const sortByName = (modules) => [...modules].sort((a, b) => a.name.localeCompare(b.name))

const writeFiles = (contents) => {
  const dir = mkdtempSync(join(tmpdir(), 'coverage-merge-'))
  const files = contents.map((content, index) => {
    const file = join(dir, `${index}.json`)
    writeFileSync(file, content)
    return file
  })
  return { dir, files }
}

test('the Go and TypeScript modules together are every measured module', () => {
  const go = collect.listCoverageModules('go')
  const ts = collect.listCoverageModules('ts')
  assert.ok(go.length > 0 && go.every((one) => one.kind === 'go'))
  assert.ok(ts.length > 0 && ts.every((one) => one.kind === 'ts'))
  assert.deepEqual(sortByName([...go, ...ts]), sortByName(collect.listCoverageModules()))
})

test('the merged files give the badge weighted by the statements of both kinds', () => {
  const go = [{ name: 'libs-core-kernel', kind: 'go', at: 'libs/core/kernel', total: 300, covered: 270 }]
  const ts = [{ name: 'libs-ui-components', kind: 'ts', at: 'libs/ui/components', total: 100, covered: 70 }]
  const { dir, files } = writeFiles([
    collect.formatMeasuredFile('ts', ts),
    collect.formatMeasuredFile('go', go),
  ])
  try {
    const measured = collect.readMeasuredFiles(files)
    assert.deepEqual(sortByName(measured), sortByName([...go, ...ts]))
    assert.equal(collect.getCoverageBadge(measured).message, '85%')
  } finally {
    rmSync(dir, { recursive: true, force: true })
  }
})

test('the measured file of a kind holds the modules of that kind only', () => {
  const measured = [
    { name: 'libs-core-kernel', kind: 'go', at: 'libs/core/kernel', total: 3, covered: 1 },
    { name: 'plugins-notes', kind: 'ts', at: 'plugins/notes', total: 4, covered: 2 },
  ]
  assert.deepEqual(JSON.parse(collect.formatMeasuredFile('go', measured)), {
    kind: 'go',
    measured: [measured[0]],
  })
})

test('the merge fails when a kind is missing or repeated', () => {
  const { dir, files } = writeFiles([
    collect.formatMeasuredFile('go', []),
    collect.formatMeasuredFile('go', []),
  ])
  try {
    assert.throws(() => collect.readMeasuredFiles([files[0]]), /one file of each kind/)
    assert.throws(() => collect.readMeasuredFiles(files), /one file of each kind/)
  } finally {
    rmSync(dir, { recursive: true, force: true })
  }
})

test('the options name the kind and its output file, or the files to merge and the quality file', () => {
  assert.deepEqual(collect.readOptions(['--kind', 'go', '--out', 'go.json']), {
    kind: 'go',
    out: 'go.json',
    merge: null,
    quality: null,
  })
  assert.deepEqual(collect.readOptions(['--merge', 'go.json', 'ts.json', '--quality', 'q.json']), {
    kind: null,
    out: null,
    merge: ['go.json', 'ts.json'],
    quality: 'q.json',
  })
  assert.deepEqual(collect.readOptions(['--quality', 'q.json']), {
    kind: null,
    out: null,
    merge: null,
    quality: 'q.json',
  })
})

test('the options reject an unknown kind, a kind without an output file and a merge without files', () => {
  assert.throws(() => collect.readOptions(['--kind', 'rust', '--out', 'x.json']), /--kind takes go or ts/)
  assert.throws(() => collect.readOptions(['--kind', 'ts']), /--kind needs --out/)
  assert.throws(() => collect.readOptions(['--kind', 'ts', '--out', 'x', '--merge', 'a']), /exclude each other/)
  assert.throws(() => collect.readOptions(['--kind', 'ts', '--out', 'x', '--quality', 'q']), /goes with --merge/)
  assert.throws(() => collect.readOptions(['--merge', '--quality', 'q.json']), /--merge needs the files/)
})
