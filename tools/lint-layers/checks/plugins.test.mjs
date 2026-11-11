import assert from 'node:assert/strict'
import { test } from 'node:test'
import { getGoReferences } from './helpers.mjs'
import {
  MAX_SERVICE_LINES,
  findOversizedServices,
  findServiceMachineAccess,
  findPluginEntryViolations,
  findPluginSliceShapeViolations,
  lintPlantedFrontend,
  listPluginSlices,
  parseEnabledGoLinters,
  PLANTED_SLICE_FRONTEND,
  PLANTED_SLICE_REFUSALS,
  readPluginPackageExports,
} from './plugins.mjs'

const EXPORTS = {
  '@numen/plugin-sdk': {
    '.': './src/index.ts',
    './testing': './src/testing/index.ts',
    './vite': './src/vite.ts',
    './internal': './src/internal.ts',
  },
  '@numen/ui': { '.': { import: './dist/index.js' }, './styles.css': './dist/numen-ui.css' },
}

const file = (name, ...values) => ({
  file: name,
  literals: values.map((value, index) => ({ line: index + 1, value })),
})

test('the main entry, the test entry and a stylesheet are clean in a plugin source file', () => {
  const files = [
    file(
      'plugins/a/frontend/index.ts',
      '@numen/plugin-sdk',
      '@numen/plugin-sdk/testing',
      '@numen/ui',
      '@numen/ui/styles.css',
    ),
  ]
  assert.deepEqual(findPluginEntryViolations(files, EXPORTS), [])
})

test('a build entry is clean in a build configuration file', () => {
  const files = [file('plugins/a/vite.config.ts', '@numen/plugin-sdk/vite')]
  assert.deepEqual(findPluginEntryViolations(files, EXPORTS), [])
})

test('a declared build entry named from a source file is reported with the line', () => {
  const found = findPluginEntryViolations(
    [file('plugins/a/frontend/index.ts', '@numen/ui', '@numen/plugin-sdk/vite')],
    EXPORTS,
  )
  assert.equal(found.length, 1)
  assert.equal(found[0].line, 2)
  assert.match(found[0].symbol, /reaches past the main entry of @numen\/plugin-sdk/)
})

test('a deep path the package does not declare is reported, in a build configuration too', () => {
  const found = findPluginEntryViolations(
    [
      file('plugins/a/vite.config.ts', '@numen/ui/src/index.ts'),
      file('plugins/a/frontend/b.ts', '@numen/plugin-sdk/dist/agent'),
    ],
    EXPORTS,
  )
  assert.equal(found.length, 2)
  assert.match(found[0].symbol, /is not an entry @numen\/ui declares/)
})

test('a package that is not a plugin package is not examined', () => {
  assert.deepEqual(
    findPluginEntryViolations(
      [file('plugins/a/b.ts', 'vue/server-renderer', '@numen/plugin-sdkx/y')],
      EXPORTS,
    ),
    [],
  )
})

test('the exports of the plugin packages of this repository are read', () => {
  const read = readPluginPackageExports()
  assert.ok('.' in read['@numen/plugin-sdk'])
  assert.ok('.' in read['@numen/ui'])
})

test('enabled Go linters are read from the YAML structure', () => {
  const yaml = 'version: "2"\nlinters:\n  default: none\n  enable:\n    - govet\n    # comment\n    - errcheck\n'
  assert.deepEqual(parseEnabledGoLinters(yaml), ['errcheck', 'govet'])
})

const SLICED = [
  'plugins/a/frontend/index.ts',
  'plugins/a/frontend/words.ts',
  'plugins/a/frontend/types.ts',
  'plugins/a/frontend/styles.css',
  'plugins/a/frontend/api/wire.ts',
  'plugins/a/frontend/ui/Tab.vue',
  'plugins/a/frontend/deck/index.ts',
  'plugins/a/frontend/deck/ui/Deck.vue',
  'plugins/a/frontend/deck/model/useDeck.ts',
  'plugins/a/frontend/preset/index.ts',
  'plugins/a/frontend/preset/lib/curve.ts',
  'plugins/b/frontend/index.ts',
  'plugins/b/frontend/ui/Tab.vue',
]

test('a frontend of the entry, the shared files, root segments and slices of segments keeps its shape', () => {
  assert.deepEqual(findPluginSliceShapeViolations(SLICED), [])
  assert.deepEqual(listPluginSlices(SLICED), ['a/deck', 'a/preset'])
})

test('tests, stories and test support are not part of the shape', () => {
  const files = [
    ...SLICED,
    'plugins/a/frontend/kind.test.ts',
    'plugins/a/frontend/Tab.stories.ts',
    'plugins/a/frontend/fixtures/host.ts',
    'plugins/a/frontend/deck/fixtures/deck.ts',
    'plugins/a/frontend/deck/testdata/deck.json',
    'plugins/a/frontend/testing/noLayout.ts',
  ]
  assert.deepEqual(findPluginSliceShapeViolations(files), [])
  assert.deepEqual(listPluginSlices(files), ['a/deck', 'a/preset'])
})

test('a file at the frontend root besides the entry, the words, the types and the stylesheet is reported', () => {
  const found = findPluginSliceShapeViolations([...SLICED, 'plugins/a/frontend/kind.ts', 'plugins/a/frontend/Tab.vue'])
  assert.deepEqual(
    found.map((one) => one.file),
    ['plugins/a/frontend/kind.ts', 'plugins/a/frontend/Tab.vue'],
  )
})

test('a slice file outside the segments, a folder that is no segment and a slice with no index are reported', () => {
  const found = findPluginSliceShapeViolations([
    ...SLICED,
    'plugins/a/frontend/deck/kind.ts',
    'plugins/a/frontend/deck/wire/errors.ts',
    'plugins/a/frontend/review/ui/Review.vue',
  ])
  assert.deepEqual(
    found.map((one) => one.file),
    ['plugins/a/frontend/deck/kind.ts', 'plugins/a/frontend/deck/wire/errors.ts', 'plugins/a/frontend/review'],
  )
  assert.match(found[2].symbol, /the slice review has no index.ts/)
})

test('the slice rule of the golden configuration refuses a deep import and a sibling slice in a planted frontend', () => {
  assert.deepEqual(lintPlantedFrontend(PLANTED_SLICE_FRONTEND), PLANTED_SLICE_REFUSALS)
})

test('the slice rule of the golden configuration passes a frontend that reaches a slice through its index', () => {
  const files = {
    ...PLANTED_SLICE_FRONTEND,
    'frontend/index.ts': "import { alpha } from './alpha'\nexport const both = [alpha]\n",
    'frontend/beta/ui/count.ts': "import { shared } from '../../lib/shared'\nexport const count = shared\n",
    'frontend/lib/shared.ts': 'export const shared = 2\n',
  }
  assert.deepEqual(lintPlantedFrontend(files), {})
})

test('a service within the limit is clean', () => {
  const backends = [{ plugin: 'a', serviceLines: MAX_SERVICE_LINES, innerPackages: [] }]
  assert.deepEqual(findOversizedServices(backends), [])
})

test('a service over the limit is reported with the inner packages it lacks', () => {
  const backends = [
    { plugin: 'big', serviceLines: MAX_SERVICE_LINES + 1, innerPackages: ['api', 'domain'] },
    {
      plugin: 'split',
      serviceLines: MAX_SERVICE_LINES + 1,
      innerPackages: ['domain', 'usecase/read', 'ports'],
    },
  ]
  const found = findOversizedServices(backends)
  assert.equal(found.length, 2)
  assert.match(found[0].symbol, /no usecase, ports package/)
  assert.doesNotMatch(found[1].symbol, /; the backend has no/)
})

test('a service source with machine access imports is refused', () => {
  const source = [
    'package service',
    '',
    'import (',
    '\t"net/http"',
    '\t"os"',
    '\t"sync"',
    ')',
    '',
    'var _ = http.StatusOK',
  ].join('\n')
  const found = findServiceMachineAccess([
    { file: 'plugins/a/backend/service/x.go', imports: getGoReferences(source).imports },
  ])
  assert.deepEqual(
    found.map((one) => one.symbol.split(';')[0]),
    ['service imports os', 'service imports sync'],
  )
})

test('a service source with plain imports is accepted', () => {
  const source = 'package service\n\nimport (\n\t"context"\n\t"net/http"\n)\n'
  assert.deepEqual(
    findServiceMachineAccess([
      { file: 'plugins/a/backend/service/x.go', imports: getGoReferences(source).imports },
    ]),
    [],
  )
})
