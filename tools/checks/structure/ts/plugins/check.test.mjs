import assert from 'node:assert/strict'
import { test } from 'node:test'
import { getGoReferences } from '../../../lib/graph.mjs'
import {
  MAX_SERVICE_LINES,
  findOversizedServices,
  findPluginBackendLayerInversions,
  findServiceImportsOffAllowList,
  findOversizedBackendPackages,
  MAX_FILE_LINES,
  findPluginEntryViolations,
  findPluginSliceShapeViolations,
  lintPlantedFrontend,
  listPluginSlices,
  parseEnabledGoLinters,
  findGoLinterSettingsDrift,
  PLANTED_SLICE_FRONTEND,
  PLANTED_SLICE_REFUSALS,
  readPluginPackageExports,
} from './check.mjs'

const EXPORTS = {
  '@numen/plugin-sdk': {
    '.': './src/index.ts',
    './testing': './src/testing/index.ts',
    './contracts': './src/contracts/index.ts',
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

test('the contracts entry is clean in a plugin source file, a path below it is reported', () => {
  const found = findPluginEntryViolations(
    [
      file(
        'plugins/a/frontend/index.ts',
        '@numen/plugin-sdk/contracts',
        '@numen/plugin-sdk/contracts/acp',
      ),
    ],
    EXPORTS,
  )
  assert.equal(found.length, 1)
  assert.equal(found[0].line, 2)
  assert.match(found[0].symbol, /is not an entry @numen\/plugin-sdk declares/)
})

test('a contracts entry of a package other than the SDK is reported', () => {
  const exports = {
    ...EXPORTS,
    '@numen/ui': { ...EXPORTS['@numen/ui'], './contracts': './dist/contracts.js' },
  }
  const found = findPluginEntryViolations(
    [file('plugins/a/frontend/index.ts', '@numen/ui/contracts')],
    exports,
  )
  assert.equal(found.length, 1)
  assert.match(found[0].symbol, /reaches past the main entry of @numen\/ui/)
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
  const yaml =
    'version: "2"\nlinters:\n  default: none\n  enable:\n    - govet\n    # comment\n    - errcheck\n'
  assert.deepEqual(parseEnabledGoLinters(yaml), ['errcheck', 'govet'])
})

test('linter settings that differ from the root are reported, the import rules and aliases of the SDK are not', () => {
  const root = [
    'linters:',
    '  settings:',
    '    gocritic:',
    '      disabled-checks: [argOrder]',
    '    depguard:',
    '      rules: {core: {deny: [{pkg: os}]}}',
    '    importas:',
    '      alias: [{pkg: a, alias: b}]',
    '    usetesting:',
    '      os-temp-dir: true',
  ].join('\n')
  const sdk = [
    'linters:',
    '  settings:',
    '    gocritic:',
    '      disabled-checks: []',
    '    depguard:',
    '      rules: {sdk: {deny: [{pkg: net}]}}',
    '    importas:',
    '      alias: []',
    '    staticcheck:',
    '      checks: [all]',
  ].join('\n')

  assert.deepEqual(
    findGoLinterSettingsDrift(root, sdk, 'sdk/.golangci.yml').map((one) => one.symbol),
    [
      "settings of linter 'gocritic' differ from root .golangci.yml",
      "settings of linter 'staticcheck' differ from root .golangci.yml",
      "settings of linter 'usetesting' differ from root .golangci.yml",
    ],
  )
  assert.deepEqual(findGoLinterSettingsDrift(root, root, 'sdk/.golangci.yml'), [])
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
  const found = findPluginSliceShapeViolations([
    ...SLICED,
    'plugins/a/frontend/kind.ts',
    'plugins/a/frontend/Tab.vue',
  ])
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
    [
      'plugins/a/frontend/deck/kind.ts',
      'plugins/a/frontend/deck/wire/errors.ts',
      'plugins/a/frontend/review',
    ],
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
    'frontend/beta/ui/count.ts':
      "import { shared } from '../../lib/shared'\nexport const count = shared\n",
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

const serviceImports = (plugin, imports) => {
  const source = `package service\n\nimport (\n${imports.map((one) => `\t"${one}"\n`).join('')})\n`
  return findServiceImportsOffAllowList([
    { file: `plugins/${plugin}/backend/service/x.go`, imports: getGoReferences(source).imports },
  ]).map((one) => one.symbol.split(',')[0])
}

test('a service source imports only the allow list: machine access, state and other packages are refused', () => {
  const own = 'github.com/jiva-studio/numen/plugins/a/backend'
  assert.deepEqual(
    serviceImports('a', [
      'os',
      'sync',
      'log',
      'io',
      'net',
      'os/exec',
      'github.com/google/uuid',
      `${own}/internal/vaulttest`,
      `${own}/internal`,
      `${own}/cards`,
      'github.com/jiva-studio/numen/plugins/b/backend/api',
      'github.com/jiva-studio/numen/plugins/b/backend/wire/x',
      'github.com/jiva-studio/numen/plugins/b/backend/internal/adapter/files',
      'github.com/jiva-studio/numen/libs/core/kernel/container',
    ]),
    [
      'service imports os',
      'service imports sync',
      'service imports log',
      'service imports io',
      'service imports net',
      'service imports os/exec',
      'service imports github.com/google/uuid',
      `service imports ${own}/internal/vaulttest`,
      `service imports ${own}/internal`,
      `service imports ${own}/cards`,
      'service imports github.com/jiva-studio/numen/plugins/b/backend/api',
      'service imports github.com/jiva-studio/numen/plugins/b/backend/wire/x',
      'service imports github.com/jiva-studio/numen/plugins/b/backend/internal/adapter/files',
      'service imports github.com/jiva-studio/numen/libs/core/kernel/container',
    ],
  )
})

test('a service source with listed standard packages, SDK packages and its own api, domain, usecase, ports, wire and adapters is accepted', () => {
  const own = 'github.com/jiva-studio/numen/plugins/a/backend'
  assert.deepEqual(
    serviceImports('a', [
      'context',
      'errors',
      'fmt',
      'net/http',
      'time',
      'encoding/json',
      'github.com/jiva-studio/numen/libs/plugin/sdk-go/server',
      'github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain',
      `${own}/api`,
      `${own}/domain`,
      `${own}/usecase/links`,
      `${own}/ports`,
      `${own}/wire/cardwire`,
      `${own}/internal/adapter/hostclient`,
    ]),
    [],
  )
})

test('a backend file over the file limit and a package over the package limit are reported, tests are not counted by the caller', () => {
  const lines = (file, count) => ({ file, lines: count })
  assert.deepEqual(
    findOversizedBackendPackages([lines('plugins/a/backend/x/a.go', MAX_FILE_LINES)]),
    [],
  )
  const long = findOversizedBackendPackages([lines('plugins/a/backend/x/a.go', MAX_FILE_LINES + 1)])
  assert.deepEqual(
    long.map((one) => [one.file, one.symbol.split(',')[0]]),
    [['plugins/a/backend/x/a.go', `file holds ${MAX_FILE_LINES + 1} lines`]],
  )
  const many = Array.from({ length: 7 }, (_, at) => lines(`plugins/a/backend/y/f${at}.go`, 450))
  assert.deepEqual(
    findOversizedBackendPackages(many).map((one) => [one.file, one.symbol.split(',')[0]]),
    [['plugins/a/backend/y', 'package holds 3150 non-test lines']],
  )
  assert.deepEqual(
    findOversizedBackendPackages([...many.slice(0, 6), lines('plugins/a/backend/z/f.go', 450)]),
    [],
  )
})

const layerImports = (file, imports) => {
  const source = `package x\n\nimport (\n${imports.map((one) => `\t"${one}"\n`).join('')})\n`
  return findPluginBackendLayerInversions([{ file, imports: getGoReferences(source).imports }]).map(
    (one) => one.file,
  )
}

test('an inner package importing wire, and wire importing service, engine or an adapter, are refused', () => {
  const own = 'github.com/jiva-studio/numen/plugins/a/backend'
  assert.deepEqual(
    layerImports('plugins/a/backend/usecase/x/x.go', [
      `${own}/wire/cardwire`,
      `${own}/service`,
      `${own}/engine`,
    ]),
    [
      'plugins/a/backend/usecase/x/x.go',
      'plugins/a/backend/usecase/x/x.go',
      'plugins/a/backend/usecase/x/x.go',
    ],
  )
  assert.deepEqual(
    layerImports('plugins/a/backend/wire/cardwire/x.go', [
      `${own}/service`,
      `${own}/engine`,
      `${own}/internal/adapter/files`,
    ]),
    [
      'plugins/a/backend/wire/cardwire/x.go',
      'plugins/a/backend/wire/cardwire/x.go',
      'plugins/a/backend/wire/cardwire/x.go',
    ],
  )
})

test('wire imports the api, domain, use cases, ports and its sibling wire packages; the service imports wire', () => {
  const own = 'github.com/jiva-studio/numen/plugins/a/backend'
  assert.deepEqual(
    layerImports('plugins/a/backend/wire/cardwire/x.go', [
      `${own}/api`,
      `${own}/domain/format`,
      `${own}/usecase/cards`,
      `${own}/ports`,
      `${own}/wire/filewire`,
      `${own}/internal/enumtest`,
    ]),
    [],
  )
  assert.deepEqual(
    layerImports('plugins/a/backend/service/x.go', [
      `${own}/wire/cardwire`,
      `${own}/internal/adapter/files`,
    ]),
    [],
  )
})

const refusedImports = (file, imports) => {
  const source = `package x\n\nimport (\n${imports.map((one) => `\t"${one}"\n`).join('')})\n`
  return findPluginBackendLayerInversions([{ file, imports: getGoReferences(source).imports }]).map(
    (one) => one.symbol.split(' in plugins/')[0],
  )
}

test('a use case importing another use case, the api, os, the SDK server or an adapter is refused', () => {
  const own = 'github.com/jiva-studio/numen/plugins/a/backend'
  const sdk = 'github.com/jiva-studio/numen/libs/plugin/sdk-go'
  assert.deepEqual(
    refusedImports('plugins/a/backend/usecase/cards/x.go', [
      `${own}/usecase/note`,
      `${own}/usecase/cards/deckedit`,
      `${own}/api`,
      'os',
      `${sdk}/server`,
      `${own}/internal/adapter/clock`,
    ]),
    [
      `use case usecase/cards imports ${own}/usecase/note`,
      `use case usecase/cards imports ${own}/usecase/cards/deckedit`,
      `use case usecase/cards imports ${own}/api`,
      'use case usecase/cards imports os',
      `use case usecase/cards imports ${sdk}/server`,
      `use case usecase/cards imports ${own}/internal/adapter/clock`,
    ],
  )
})

test('a use case imports its ports and domain, the listed standard and SDK packages and third-party modules', () => {
  const own = 'github.com/jiva-studio/numen/plugins/a/backend'
  const sdk = 'github.com/jiva-studio/numen/libs/plugin/sdk-go'
  assert.deepEqual(
    refusedImports('plugins/a/backend/usecase/cards/deckedit/x.go', [
      'context',
      'strings',
      `${own}/ports`,
      `${own}/domain/format`,
      `${own}/epub/cfi`,
      `${sdk}/host/domain`,
      `${sdk}/contracts/llm`,
      'golang.org/x/text/unicode/norm',
    ]),
    [],
  )
  assert.deepEqual(
    refusedImports('plugins/a/backend/usecase/cards/x.go', [
      'os/exec',
      'path/filepath',
      `${sdk}/hostcall`,
      'github.com/jiva-studio/numen/plugins/b/backend/domain/format',
    ]),
    [
      'use case usecase/cards imports os/exec',
      'use case usecase/cards imports path/filepath',
      `use case usecase/cards imports ${sdk}/hostcall`,
      'use case usecase/cards imports github.com/jiva-studio/numen/plugins/b/backend/domain/format',
    ],
  )
})

test('an adapter importing the api, wire, service, engine or another technology is refused', () => {
  const own = 'github.com/jiva-studio/numen/plugins/a/backend'
  assert.deepEqual(
    refusedImports('plugins/a/backend/internal/adapter/rpc/x.go', [
      `${own}/api`,
      `${own}/wire/cardwire`,
      `${own}/service`,
      `${own}/engine`,
      `${own}/internal/adapter/clock`,
      `${own}/internal/vaulttest`,
    ]),
    [
      `adapter internal/adapter/rpc imports ${own}/api`,
      `adapter internal/adapter/rpc imports ${own}/wire/cardwire`,
      `adapter internal/adapter/rpc imports ${own}/service`,
      `adapter internal/adapter/rpc imports ${own}/engine`,
      `adapter internal/adapter/rpc imports ${own}/internal/adapter/clock`,
      `adapter internal/adapter/rpc imports ${own}/internal/vaulttest`,
    ],
  )
})

test('an adapter imports the machine, the SDK, its ports, domain and use case, and its own technology folder', () => {
  const own = 'github.com/jiva-studio/numen/plugins/a/backend'
  assert.deepEqual(
    refusedImports('plugins/a/backend/internal/adapter/engine/segmenter/x.go', [
      'os',
      'os/exec',
      'github.com/jiva-studio/numen/libs/plugin/sdk-go/server',
      `${own}/ports`,
      `${own}/domain/settings`,
      `${own}/pdf`,
      `${own}/usecase/transcribe`,
      `${own}/internal/adapter/engine`,
      `${own}/internal/adapter/engine/onnx`,
    ]),
    [],
  )
})
