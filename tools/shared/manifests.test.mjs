/**
 * How manifests are read: against the generated schema, from fixtures written here and from the tree.
 */
import assert from 'node:assert/strict'
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import test from 'node:test'
import {
  ManifestError,
  PLUGINS_DIR,
  listPluginNames,
  parseManifest,
  readPluginManifests,
} from './manifests.mjs'

const valid = {
  id: 'sample.plugin',
  name: 'Sample',
  version: '1.0.0',
  targets: ['desktop.editor'],
  contributes: {
    feature: 'Samples',
    category: 'tools',
    tabKinds: [{ kind: 'sample', title: 'Sample' }],
  },
}

const withPlugins = (fill, check) => {
  const root = mkdtempSync(join(tmpdir(), 'manifests-'))
  try {
    fill(root)
    check(root)
  } finally {
    rmSync(root, { recursive: true, force: true })
  }
}

const writePlugin = (root, name, text) => {
  mkdirSync(join(root, name), { recursive: true })
  if (text !== null) writeFileSync(join(root, name, 'manifest.json'), text)
}

test('a manifest is read as the schema message and as the document', () => {
  const { manifest, document } = parseManifest(JSON.stringify(valid))

  assert.equal(manifest.id, 'sample.plugin')
  assert.equal(manifest.contributes.feature, 'Samples')
  assert.equal(manifest.contributes.tabKinds[0].kind, 'sample')
  assert.deepEqual(document, valid)
})

test('a key the schema does not name is ignored', () => {
  const { manifest } = parseManifest(JSON.stringify({ ...valid, futureField: { nested: true } }))

  assert.equal(manifest.name, 'Sample')
})

test('a value of the wrong type is refused and the error names the source', () => {
  assert.throws(
    () =>
      parseManifest(
        JSON.stringify({ ...valid, targets: 'desktop.editor' }),
        'plugins/a/manifest.json',
      ),
    (error) => {
      assert.ok(error instanceof ManifestError)
      assert.match(error.message, /plugins\/a\/manifest\.json does not fit the manifest schema/)
      return true
    },
  )
})

test('a frontend that is not an object is refused', () => {
  assert.throws(() => parseManifest(JSON.stringify({ ...valid, frontend: true })), ManifestError)
})

test('text that is not JSON, and JSON that is not an object, are refused', () => {
  assert.throws(() => parseManifest('{'), /is not JSON/)
  assert.throws(() => parseManifest('[]'), /is not a JSON object/)
  assert.throws(() => parseManifest('null'), /is not a JSON object/)
})

test('a manifest with no id, name or version is refused', () => {
  assert.throws(() => parseManifest(JSON.stringify({ id: 'a' })), /has no name, version/)
  assert.throws(() => parseManifest('{}'), /has no id, name, version/)
})

test('the plugins of a folder are the directories that hold a manifest', () =>
  withPlugins(
    (root) => {
      writePlugin(root, 'b-plugin', JSON.stringify({ ...valid, id: 'b' }))
      writePlugin(root, 'a-plugin', JSON.stringify({ ...valid, id: 'a' }))
      writePlugin(root, 'no-manifest', null)
    },
    (root) => {
      assert.deepEqual(listPluginNames(root), ['a-plugin', 'b-plugin'])
      assert.deepEqual(
        readPluginManifests(root).map((one) => [one.name, one.manifest.id]),
        [
          ['a-plugin', 'a'],
          ['b-plugin', 'b'],
        ],
      )
    },
  ))

test('a folder with an invalid manifest is refused as a whole', () =>
  withPlugins(
    (root) => {
      writePlugin(root, 'good', JSON.stringify(valid))
      writePlugin(root, 'bad', JSON.stringify({ ...valid, permissions: 3 }))
    },
    (root) => assert.throws(() => readPluginManifests(root), ManifestError),
  ))

test('the manifests of the tree fit the schema', () => {
  const plugins = readPluginManifests(PLUGINS_DIR)

  assert.ok(plugins.length >= 10, `${plugins.length} plugins read`)
  for (const one of plugins) assert.match(one.manifest.version, /^\d{4}\.\d+\.\d+$/, one.name)
})
