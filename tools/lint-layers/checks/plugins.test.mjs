import assert from 'node:assert/strict'
import { test } from 'node:test'
import { findPluginEntryViolations, readPluginPackageExports } from './plugins.mjs'

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
