/**
 * How the plugin index is built: from fixtures written here and from the plugins of the repository.
 */
import assert from 'node:assert/strict'
import { mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import test from 'node:test'
import { readPluginManifests } from '../shared/manifests.mjs'
import { INDEX_FILE, buildIndex, formatIndex, writeIndex } from './build.mjs'

const sample = (id, extra = {}) => ({
  id,
  name: id,
  version: '1.0.0',
  targets: ['desktop.editor'],
  contributes: {
    feature: 'Samples',
    category: 'tools',
    tabKinds: [{ kind: 'sample', title: 'S' }],
  },
  ...extra,
})

const withPlugins = (plugins, check) => {
  const root = mkdtempSync(join(tmpdir(), 'plugin-index-'))
  try {
    for (const [name, document] of Object.entries(plugins)) {
      mkdirSync(join(root, name), { recursive: true })
      writeFileSync(join(root, name, 'manifest.json'), JSON.stringify(document))
    }
    check(root)
  } finally {
    rmSync(root, { recursive: true, force: true })
  }
}

test('the index lists the manifest of every plugin ordered by id', () => {
  withPlugins({ b: sample('org.example.b'), a: sample('org.example.z') }, (root) => {
    const index = buildIndex(root)

    assert.deepEqual(
      index.plugins.map((plugin) => plugin.id),
      ['org.example.b', 'org.example.z'],
    )
    assert.equal(index.plugins[0].contributes.tabKinds[0].kind, 'sample')
  })
})

test('a key the schema does not name is left out of the index', () => {
  withPlugins({ a: sample('org.example.a', { futureField: true }) }, (root) => {
    assert.equal('futureField' in buildIndex(root).plugins[0], false)
  })
})

test('a folder with no plugin is an error', () => {
  withPlugins({}, (root) => {
    assert.throws(() => buildIndex(root), /no plugin with a manifest\.json/)
  })
})

test('a plugin id declared twice is an error', () => {
  withPlugins({ a: sample('org.example.a'), b: sample('org.example.a') }, (root) => {
    assert.throws(() => buildIndex(root), /declared twice/)
  })
})

test('a manifest that does not fit the schema is an error naming the file', () => {
  withPlugins({ a: sample('org.example.a', { targets: 'desktop.editor' }) }, (root) => {
    assert.throws(() => buildIndex(root), /manifest\.json/)
  })
})

test('the index is written next to the plugins and is read back unchanged', () => {
  withPlugins({ a: sample('org.example.a') }, (root) => {
    const index = writeIndex(root)

    assert.equal(readFileSync(join(root, INDEX_FILE), 'utf8'), formatIndex(index))
  })
})

test('the index of the repository holds every plugin and the files they open', () => {
  const manifests = readPluginManifests()
  const index = buildIndex()

  assert.ok(
    manifests.length >= 15,
    `only ${manifests.length} plugins: the rule is checked against too little`,
  )
  assert.equal(index.plugins.length, manifests.length)
  assert.deepEqual(
    index.plugins.map((plugin) => plugin.id).sort(),
    manifests.map(({ manifest }) => manifest.id).sort(),
  )
  const opening = index.plugins.filter((plugin) => (plugin.contributes?.fileTypes ?? []).length > 0)
  assert.ok(opening.length >= 4, `only ${opening.length} plugins open files`)
})
