/**
 * How a run is split, how its mutants are named, and what is done with the
 * target's issue.
 */
import assert from 'node:assert/strict'
import test from 'node:test'
import { getGoShardOf, keepFiles, mergeGoCaches, mergeStrykerReports } from './cache.mjs'
import { groupByLine, isTargetIssue, planIssue, readCount, readKeys, renderBody } from './issue.mjs'
import { findTarget, formatSurvivors, getRunName, renderVitestConfig } from './mutate.mjs'
import { patchSource } from './patch.mjs'
import { listShards } from './shards.mjs'
import { getOriginal, readMutants, summarize } from './report.mjs'
import {
  findTestdata,
  getShardCount,
  getShardOf,
  getTargets,
  isMutatedGo,
  isMutatedSource,
  keepOnly,
  nameTargets,
  pickShard,
} from './targets.mjs'

const go = { name: 'libs-core', kind: 'go', at: 'libs/core', shards: 1, testdata: [] }
const ts = { name: 'libs-ui', kind: 'ts', at: 'libs/ui/core', shards: 1, testdata: [] }

const at = (line, column, endColumn) => ({
  start: { line, column },
  end: { line, column: endColumn },
})

const report = (source, mutants, key = 'libs/ui/core/src/lib/span.ts') => ({
  files: { [key]: { source, mutants } },
})

const link = (one) => `https://example.test/${one.path}#L${one.line}`

/* --------------------------------- Shards --------------------------------- */

test('every item lands in exactly one shard', () => {
  const items = Array.from({ length: 40 }, (_, index) => `./pkg/item-${index}`)
  const shards = [1, 2, 3, 4].map((index) => pickShard(items, index, 4))
  assert.deepEqual(shards.flat().sort(), [...items].sort())
  assert.ok(shards.every((one) => one.length > 0))
})

test('an added item moves no other between shards', () => {
  const items = Array.from({ length: 40 }, (_, index) => `./pkg/item-${index}`)
  const before = pickShard(items, 2, 4)
  const after = pickShard([...items, './pkg/added'], 2, 4)
  assert.deepEqual(
    after.filter((one) => one !== './pkg/added'),
    before,
  )
})

test('a target takes a shard per slice of its source, within the ceiling', () => {
  assert.equal(getShardCount('go', 1), 1)
  assert.equal(getShardCount('go', 6001), 2)
  assert.equal(getShardCount('ts', 4500), 1)
  assert.equal(getShardCount('go', 10_000_000), 16)
})

test('a folder holding a Go module and a package names both by kind', () => {
  const named = nameTargets([
    { kind: 'go', at: 'apps/mobile' },
    { kind: 'ts', at: 'apps/mobile' },
    { kind: 'go', at: 'libs/core' },
  ])
  assert.deepEqual(
    named.map((one) => one.name),
    ['apps-mobile-go', 'apps-mobile-ts', 'libs-core'],
  )
})

test('test data outside the package is found from the paths its tests import', () => {
  const tests = [
    {
      path: 'libs/ui/core/src/shared/lib/text/address.test.ts',
      source:
        "import cases from '../../../../../../protocol/testdata/addresses.json'\nimport { read } from './address'\nimport raw from '../../../../../../protocol/testdata/html/spine.html?raw'",
    },
  ]
  assert.deepEqual(findTestdata('libs/ui/core', tests), ['libs/protocol/testdata'])
})

test('the tree yields the core module and the components package, with what their tests read', () => {
  const found = getTargets()
  const core = found.find((one) => one.at === 'libs/core')
  const ui = found.find((one) => one.at === 'libs/ui/core')
  assert.equal(core?.kind, 'go')
  assert.ok(core.shards > 1)
  assert.equal(ui?.kind, 'ts')
  assert.ok(ui.testdata.includes('libs/protocol/testdata'))
  assert.equal(new Set(found.map((one) => one.name)).size, found.length)
})

test('the generated Vitest config keeps the projects that do not run in a browser', () => {
  const config = renderVitestConfig('apps/desktop/editor')
  assert.match(config, /from '\.\.\/\.\.\/apps\/desktop\/editor\/vitest\.config\.ts'/)
  assert.match(config, /!isBrowser\(one\)/)
})

test('source under test is told apart from tests, stories and barrels', () => {
  assert.equal(isMutatedSource('libs/ui/core/src/lib/span.ts'), true)
  assert.equal(isMutatedSource('libs/ui/core/src/lib/span.test.ts'), false)
  assert.equal(isMutatedSource('libs/ui/core/src/lib/span.stories.ts'), false)
  assert.equal(isMutatedSource('libs/ui/core/src/lib/index.ts'), false)
  assert.equal(isMutatedSource('apps/desktop/editor/src/main.ts'), false)
  assert.equal(isMutatedSource('libs/ui/core/src/lib/Span.vue'), false)
  assert.equal(isMutatedGo('usecase/note/open.go'), true)
  assert.equal(isMutatedGo('usecase/note/open_test.go'), false)
  assert.equal(isMutatedGo('usecase/note/testdata/case.go'), false)
  assert.equal(isMutatedGo('internal/testsupport/indexfile/index.go'), false)
  assert.equal(isMutatedSource('apps/desktop/editor/src/testing/answers.ts'), false)
  assert.equal(isMutatedSource('libs/ui/core/src/features/plex/fixtures/build.ts'), false)
  assert.equal(isMutatedSource('libs/ui/core/src/shared/fixtures/clock.ts'), false)
  assert.equal(isMutatedGo('usecase/note/fixtures/case.go'), false)
})

test('the sandbox keeps the package and its test data and drops their siblings', () => {
  assert.deepEqual(keepOnly(['libs/ui/core', 'libs/protocol/testdata']), [
    '/*',
    '!/libs',
    '/libs/*',
    '!/libs/ui',
    '/libs/ui/*',
    '!/libs/ui/core',
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
  assert.equal(before[0].path, 'libs/ui/core/src/lib/span.ts')
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
  assert.equal(one.path, 'libs/core/internal/cardid/cardid.go')
})

test('efficacy counts killed and timed-out mutants against the survivors only', () => {
  const statuses = ['Killed', 'Killed', 'Timeout', 'Survived', 'NoCoverage', 'CompileError']
  const summary = summarize(statuses.map((status) => ({ status })))
  assert.equal(summary.efficacy, 75)
  assert.equal(summary.total, 6)
  assert.equal(summary.counts.CompileError, 1)
})

/* ---------------------------------- Issue --------------------------------- */

const survivor = (key) => ({
  key,
  path: 'libs/ui/core/src/lib/span.ts',
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

test('the runner patch joins the suite chain as Vitest 5 does, once', () => {
  const source = "function collectTestName() {\n    return nameParts.join(' ').trim();\n}"
  const patched = patchSource(source)
  assert.match(patched, /join\(' > '\)/)
  assert.equal(patchSource(patched), patched)
  assert.throws(() => patchSource('function collectTestName() {}'))
})

test('the matrix names each shard as the run names its report', () => {
  const targets = getTargets()
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

test('the body lists one survivor per line and counts the rest of that line', () => {
  const on = (key, line, path = 'libs/ui/core/src/lib/span.ts') => ({
    ...survivor(key),
    line,
    path,
  })
  const lines = groupByLine([on('cccccc3333', 9), on('aaaaaa1111', 4), on('bbbbbb2222', 4)])
  assert.deepEqual(
    lines.map(({ survivor: one, more }) => [one.line, more]),
    [
      [4, 1],
      [9, 0],
    ],
  )
})

test('the body carries how many survived, for the next run to compare', () => {
  const body = renderBody({
    name: ts.name,
    summary: { counts: { Survived: 2 }, total: 2, efficacy: 0 },
    survivors: [survivor('aaaaaa1111'), survivor('bbbbbb2222')],
    link,
    runUrl: 'https://example.test/runs/2',
    previous: 5,
  })
  assert.equal(readCount(body), 2)
  assert.match(body, /Survivors: 2 \(5 on the run before\), on 1 lines/)
  assert.equal(readCount('no marker'), null)
})

/* ---------------------------------- File ---------------------------------- */

test('a file is mutated with the deepest target of its kind that holds it', () => {
  const targets = [
    { name: 'apps-desktop', kind: 'go', at: 'apps/desktop' },
    { name: 'apps-desktop-editor', kind: 'ts', at: 'apps/desktop/editor' },
    { name: 'apps-mobile-go', kind: 'go', at: 'apps/mobile' },
    { name: 'apps-mobile-ts', kind: 'ts', at: 'apps/mobile' },
  ]
  assert.equal(
    findTarget(targets, 'apps/desktop/editor/src/a.ts').name,
    'apps-desktop-editor',
  )
  assert.equal(findTarget(targets, 'apps/desktop/internal/a.go').name, 'apps-desktop')
  assert.equal(findTarget(targets, 'apps/mobile/bind/a.go').name, 'apps-mobile-go')
  assert.equal(findTarget(targets, 'apps/mobile/src/a.ts').name, 'apps-mobile-ts')
  assert.throws(() => findTarget(targets, 'libs/wire/a.ts'))
})

test('the mutants a file run prints are the unnoticed ones, in line order', () => {
  const one = (line, status) => ({ ...survivor('aaaaaa1111'), line, status })
  assert.deepEqual(formatSurvivors([one(9, 'Survived'), one(3, 'Killed'), one(4, 'NoCoverage')]), [
    'libs/ui/core/src/lib/span.ts:4 not covered EqualityOperator: a > b → a >= b',
    'libs/ui/core/src/lib/span.ts:9 survived EqualityOperator: a > b → a >= b',
  ])
})

/* ---------------------------------- Cache --------------------------------- */

/** Module-relative Go files, the first held by shard 1 of 2 and the second by shard 2. */
const findGoFiles = () => {
  const files = Array.from({ length: 20 }, (_, index) => `pkg${index}/a.go`)
  return [1, 2].map((index) => files.find((file) => getGoShardOf(file, 2) === index))
}

const goCache = (entries, toolchain = 'go1.26.0') => ({
  schema_version: 7,
  go_module: 'example.test/core',
  tool_version: 'v0.6.1',
  go_toolchain: toolchain,
  coverage_key: 'key',
  coverage_profile: 'mode: set',
  entries,
})

const entry = (rel_file, status) => ({ rel_file, line: 1, col: 1, type: 'NEGATE', status })

test('a Go cache takes each package from the shard that holds it', () => {
  const [first, second] = findGoFiles()
  const previous = goCache([entry(first, 'LIVED'), entry(second, 'LIVED')])
  const merged = mergeGoCaches(2, previous, [
    { index: 1, cache: goCache([entry(first, 'KILLED'), entry(second, 'LIVED')]) },
    { index: 2, cache: goCache([entry(first, 'LIVED'), entry(second, 'KILLED')]) },
  ])
  assert.deepEqual(
    merged.entries.map((one) => [one.rel_file, one.status]).sort(),
    [
      [first, 'KILLED'],
      [second, 'KILLED'],
    ].sort(),
  )
  assert.equal(merged.coverage_key, undefined)
  assert.equal(merged.coverage_profile, undefined)
  assert.equal(merged.go_toolchain, 'go1.26.0')
})

test('a Go shard that wrote nothing keeps its packages from the previous cache', () => {
  const [first, second] = findGoFiles()
  const previous = goCache([entry(first, 'LIVED'), entry(second, 'LIVED')])
  const merged = mergeGoCaches(2, previous, [
    { index: 1, cache: goCache([entry(first, 'KILLED')]) },
  ])
  assert.deepEqual(
    merged.entries.map((one) => [one.rel_file, one.status]).sort(),
    [
      [first, 'KILLED'],
      [second, 'LIVED'],
    ].sort(),
  )
})

test('a Go cache written under another toolchain is dropped from the merge', () => {
  const [first, second] = findGoFiles()
  const previous = goCache([entry(second, 'LIVED')], 'go1.25.0')
  const merged = mergeGoCaches(2, previous, [
    { index: 1, cache: goCache([entry(first, 'KILLED')]) },
  ])
  assert.deepEqual(
    merged.entries.map((one) => one.rel_file),
    [first],
  )
  assert.equal(mergeGoCaches(2, null, []), null)
})

/** Repository-relative TypeScript files, the first held by shard 1 of 2 and the second by shard 2. */
const findTsFiles = () => {
  const files = Array.from({ length: 20 }, (_, index) => `libs/ui/core/src/lib/file${index}.ts`)
  return [1, 2].map((index) => files.find((file) => getShardOf(file, 2) === index))
}

const TESTS = 'libs/ui/core/src/lib/span.test.ts'
const TEST_SOURCE = "it('grows', () => {})\nit('shrinks', () => {})\n"

/** A Stryker report whose tests are numbered in the order given. */
const strykerReport = (names, files, testSource = TEST_SOURCE) => ({
  schemaVersion: '1.0',
  files,
  testFiles: {
    [TESTS]: {
      source: testSource,
      tests: names.map((name, id) => ({
        id: String(id),
        name,
        location: { start: { line: name === 'grows' ? 1 : 2, column: 1 } },
      })),
    },
  },
})

const strykerFile = (status, killedBy) => ({
  source: 'export const a = 1\n',
  mutants: [{ id: '1', mutatorName: 'NumericLiteral', status, killedBy, coveredBy: killedBy }],
})

const getKillers = (report, file) =>
  report.files[file].mutants[0].killedBy.map(
    (id) => report.testFiles[TESTS].tests.find((one) => one.id === id).name,
  )

test('a Stryker report takes each file from the shard that holds it and renames its tests', () => {
  const [first, second] = findTsFiles()
  const merged = mergeStrykerReports(
    2,
    null,
    [
      {
        index: 1,
        report: strykerReport(['grows', 'shrinks'], {
          [first]: strykerFile('Killed', ['1']),
          [second]: strykerFile('Survived', []),
        }),
      },
      {
        index: 2,
        report: strykerReport(['shrinks', 'grows'], { [second]: strykerFile('Killed', ['1']) }),
      },
    ],
    () => TEST_SOURCE,
  )
  assert.equal(merged.files[second].mutants[0].status, 'Killed')
  assert.deepEqual(getKillers(merged, first), ['shrinks'])
  assert.deepEqual(getKillers(merged, second), ['grows'])
  assert.equal(merged.testFiles[TESTS].tests.length, 2)
})

test('a Stryker shard that wrote nothing keeps its files from the previous report', () => {
  const [first, second] = findTsFiles()
  const previous = strykerReport(['grows', 'shrinks'], {
    [first]: strykerFile('Survived', []),
    [second]: strykerFile('Killed', ['0']),
  })
  const merged = mergeStrykerReports(
    2,
    previous,
    [
      {
        index: 1,
        report: strykerReport(['grows', 'shrinks'], { [first]: strykerFile('Killed', ['1']) }),
      },
    ],
    () => TEST_SOURCE,
  )
  assert.equal(merged.files[first].mutants[0].status, 'Killed')
  assert.deepEqual(getKillers(merged, second), ['grows'])
})

test('a test whose file changed since a report was written is dropped with the mutants that name it', () => {
  const [, second] = findTsFiles()
  const before = "it('grows', () => { expect(1) })\nit('shrinks', () => {})\n"
  const previous = strykerReport(['grows'], { [second]: strykerFile('Killed', ['0']) }, before)
  const merged = mergeStrykerReports(2, previous, [], () => TEST_SOURCE)
  assert.deepEqual(merged.files[second].mutants[0].killedBy, [])
  assert.equal(merged.testFiles[TESTS], undefined)
})

test('a shard report keeps the files the shard mutated', () => {
  const report = { files: { 'a.ts': {}, 'b.ts': {} }, testFiles: {} }
  assert.deepEqual(Object.keys(keepFiles(report, ['b.ts']).files), ['b.ts'])
})
