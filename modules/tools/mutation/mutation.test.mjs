/**
 * How a run is split, how its mutants are named, and what is done with the
 * target's issue.
 */
import assert from 'node:assert/strict'
import test from 'node:test'
import { isTargetIssue, planIssue, readKeys, renderBody } from './issue.mjs'
import { getRunName } from './mutate.mjs'
import { patchSource } from './patch.mjs'
import { listShards } from './shards.mjs'
import { getOriginal, getSurvivors, isBroken, readMutants, summarize } from './report.mjs'
import { isMutatedGo, isMutatedSource, keepOnly, pickShard, targets } from './targets.mjs'

const go = targets.find((one) => one.kind === 'go')
const ts = targets.find((one) => one.kind === 'ts')

const at = (line, column, endColumn) => ({
  start: { line, column },
  end: { line, column: endColumn },
})

const report = (source, mutants, key = 'libs/ui/src/lib/span.ts') => ({
  files: { [key]: { source, mutants } },
})

const link = (one) => `https://example.test/${one.path}#L${one.line}`

/* --------------------------------- Shards --------------------------------- */

test('every item lands in exactly one shard', () => {
  const items = Array.from({ length: 23 }, (_, index) => `item-${index}`)
  const shards = [1, 2, 3, 4].map((index) => pickShard(items, index, 4))
  assert.deepEqual(shards.flat().sort(), [...items].sort())
  assert.ok(shards.every((one) => one.length >= 5))
})

test('source under test is told apart from tests, stories and barrels', () => {
  assert.equal(isMutatedSource('modules/libs/ui/src/lib/span.ts'), true)
  assert.equal(isMutatedSource('modules/libs/ui/src/lib/span.test.ts'), false)
  assert.equal(isMutatedSource('modules/libs/ui/src/lib/span.stories.ts'), false)
  assert.equal(isMutatedSource('modules/libs/ui/src/lib/index.ts'), false)
  assert.equal(isMutatedSource('modules/apps/desktop/editor/src/main.ts'), false)
  assert.equal(isMutatedSource('modules/libs/ui/src/lib/Span.vue'), false)
  assert.equal(isMutatedGo('usecase/note/open.go'), true)
  assert.equal(isMutatedGo('usecase/note/open_test.go'), false)
  assert.equal(isMutatedGo('usecase/note/testdata/case.go'), false)
})

test('the sandbox keeps the package and its test data and drops their siblings', () => {
  assert.deepEqual(keepOnly(['libs/ui', 'libs/protocol/testdata']), [
    '/*',
    '!/libs',
    '/libs/*',
    '!/libs/ui',
    '!/libs/protocol',
    '/libs/protocol/*',
    '!/libs/protocol/testdata',
  ])
})

/* --------------------------------- Report --------------------------------- */

test('the replaced text is read from the source, across lines too', () => {
  const source = 'const a = 1\nif (width > limit) {\n  grow(\n    width)\n}\n'
  assert.equal(getOriginal(source, at(2, 5, 18)), 'width > limit')
  assert.equal(
    getOriginal(source, { start: { line: 3, column: 3 }, end: { line: 4, column: 11 } }),
    'grow( width)',
  )
})

test('a mutant keeps its key when lines are added above it', () => {
  const mutant = {
    id: '1',
    mutatorName: 'EqualityOperator',
    replacement: 'width >= limit',
    status: 'Survived',
  }
  const before = readMutants(ts, [
    report('if (width > limit) {}\n', [{ ...mutant, location: at(1, 5, 18) }]),
  ])
  const after = readMutants(ts, [
    report('// a note\n\nif (width > limit) {}\n', [{ ...mutant, location: at(3, 5, 18) }]),
  ])
  assert.equal(before[0].key, after[0].key)
  assert.equal(before[0].path, 'modules/libs/ui/src/lib/span.ts')
  assert.equal(after[0].line, 3)
})

test('identical mutants in one file are told apart by their order', () => {
  const source = 'a > b\na > b\n'
  const mutants = [1, 2].map((line) => ({
    id: String(line),
    mutatorName: 'EqualityOperator',
    replacement: 'a >= b',
    status: 'Survived',
    location: at(line, 1, 6),
  }))
  const [first, second] = readMutants(ts, [report(source, mutants)])
  assert.notEqual(first.key, second.key)
})

test('a Go mutant is placed by the module path its id carries', () => {
  const [one] = readMutants(go, [
    report(
      'package cardid\n',
      [
        {
          id: 'internal/cardid/cardid.go:New:RETURN_TRUE#1',
          mutatorName: 'RETURN_TRUE',
          replacement: 'true',
          status: 'Survived',
          location: at(1, 1, 8),
        },
      ],
      'cardid.go',
    ),
  ])
  assert.equal(one.path, 'modules/libs/core/internal/cardid/cardid.go')
})

test('efficacy counts killed and timed-out mutants against the survivors only', () => {
  const statuses = ['Killed', 'Killed', 'Timeout', 'Survived', 'NoCoverage', 'CompileError']
  const summary = summarize(statuses.map((status) => ({ status })))
  assert.equal(summary.efficacy, 75)
  assert.equal(summary.total, 6)
  assert.equal(summary.counts.CompileError, 1)
})

test('a waived survivor is dropped for its own target only', () => {
  const mutants = [
    { key: 'aaaaaaaaaa', status: 'Survived' },
    { key: 'bbbbbbbbbb', status: 'Survived' },
    { key: 'cccccccccc', status: 'Killed' },
  ]
  const waivers = [
    { target: ts.name, key: 'aaaaaaaaaa', why: 'equivalent' },
    { target: 'elsewhere', key: 'bbbbbbbbbb', why: 'equivalent' },
  ]
  assert.deepEqual(
    getSurvivors(ts, mutants, waivers).map((one) => one.key),
    ['bbbbbbbbbb'],
  )
})

/* ---------------------------------- Issue --------------------------------- */

const survivor = (key) => ({
  key,
  path: 'modules/libs/ui/src/lib/span.ts',
  line: 4,
  mutator: 'EqualityOperator',
  original: 'a > b',
  replacement: 'a >= b',
})

const bodyWith = (keys) =>
  renderBody({
    name: ts.name,
    summary: { counts: { Survived: keys.length }, total: keys.length, efficacy: 0 },
    survivors: keys.map(survivor),
    link,
    runUrl: 'https://example.test/runs/1',
  })

test('the body names its target and carries the keys it was written with', () => {
  const body = bodyWith(['aaaaaa1111', 'bbbbbb2222'])
  assert.equal(isTargetIssue(ts.name, body), true)
  assert.equal(isTargetIssue('core', body), false)
  assert.deepEqual([...readKeys(body)], ['aaaaaa', 'bbbbbb'])
})

test('no issue and survivors: one is created', () => {
  assert.equal(planIssue(undefined, [survivor('aaaaaa1111')]).action, 'create')
})

test('the same survivors again: the body is rewritten and nothing is new', () => {
  const existing = { state: 'OPEN', body: bodyWith(['aaaaaa1111']) }
  assert.deepEqual(planIssue(existing, [survivor('aaaaaa1111')]), { action: 'update', fresh: [] })
})

test('a survivor not listed before is announced', () => {
  const existing = { state: 'OPEN', body: bodyWith(['aaaaaa1111']) }
  const plan = planIssue(existing, [survivor('aaaaaa1111'), survivor('cccccc3333')])
  assert.equal(plan.action, 'update')
  assert.deepEqual(
    plan.fresh.map((one) => one.key),
    ['cccccc3333'],
  )
})

test('no survivors left: an open issue is closed, a closed one is left alone', () => {
  assert.equal(planIssue({ state: 'OPEN', body: bodyWith(['aaaaaa1111']) }, []).action, 'close')
  assert.equal(planIssue({ state: 'CLOSED', body: bodyWith([]) }, []).action, 'none')
  assert.equal(planIssue(undefined, []).action, 'none')
})

test('a survivor on a closed issue reopens it', () => {
  assert.equal(
    planIssue({ state: 'CLOSED', body: bodyWith([]) }, [survivor('aaaaaa1111')]).action,
    'reopen',
  )
})

/* --------------------------------- Guards --------------------------------- */

test('a run that kills nothing while several survive is a broken runner', () => {
  assert.equal(isBroken({ Survived: 7 }), true)
  assert.equal(isBroken({ Survived: 7, Killed: 1 }), false)
  assert.equal(isBroken({ Survived: 2 }), false)
  assert.equal(isBroken({ Survived: 7, Timeout: 1 }), false)
})

test('the runner patch joins the suite chain as Vitest 5 does, once', () => {
  const source = "function collectTestName() {\n    return nameParts.join(' ').trim();\n}"
  const patched = patchSource(source)
  assert.match(patched, /join\(' > '\)/)
  assert.equal(patchSource(patched), patched)
  assert.throws(() => patchSource('function collectTestName() {}'))
})

test('the matrix names each shard as the run names its report', () => {
  for (const target of targets) {
    const shards = listShards(target.name)
    assert.equal(shards.length, target.shards)
    for (const one of shards) {
      assert.equal(one.name, getRunName(target, one.shard.split('/').map(Number)))
    }
  }
  assert.equal(
    listShards().length,
    targets.reduce((sum, one) => sum + one.shards, 0),
  )
})
