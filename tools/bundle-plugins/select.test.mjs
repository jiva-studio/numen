/**
 * Which plugins the macOS application carries: the selection must equal the plugins the index lists
 * for the target, whatever a plugin ships (a backend, a frontend, or only contributions).
 */
import assert from 'node:assert/strict'
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { dirname, join } from 'node:path'
import { spawnSync } from 'node:child_process'
import test from 'node:test'
import { fileURLToPath } from 'node:url'
import { buildIndex } from '../build-plugin-index/build.mjs'
import { PLUGINS_DIR, readPluginManifests } from '../shared/manifests.mjs'
import { listFolders } from './select.mjs'

const SCRIPT = join(dirname(fileURLToPath(import.meta.url)), 'select.mjs')
const EDITOR = 'desktop.editor'

const runSelect = (args) => spawnSync(process.execPath, [SCRIPT, ...args], { encoding: 'utf8' })

const listSelectedPlugins = (pluginsDir, target) => {
  const result = runSelect(['--target', target, '--plugins', pluginsDir])
  assert.equal(result.status, 0, result.stderr)
  return result.stdout.split('\n').filter((line) => line !== '')
}

const sample = (id, targets) => ({
  id,
  name: id,
  version: '1.0.0',
  targets,
  contributes: { feature: 'Samples', category: 'tools', tabKinds: [{ kind: id, title: 'S' }] },
})

const withTree = (files, check) => {
  const root = mkdtempSync(join(tmpdir(), 'bundle-plugins-'))
  try {
    for (const [path, content] of Object.entries(files)) {
      mkdirSync(dirname(join(root, path)), { recursive: true })
      writeFileSync(join(root, path), content)
    }
    check(root)
  } finally {
    rmSync(root, { recursive: true, force: true })
  }
}

const manifestOf = (document) => JSON.stringify(document)

const fixture = {
  'only-contributions/manifest.json': manifestOf(sample('org.example.themes', [EDITOR])),
  'only-contributions/themes/a.css': 'a {}',
  'with-bin/manifest.json': manifestOf(sample('org.example.backend', [EDITOR])),
  'with-bin/bin/tool': 'x',
  'with-dist/manifest.json': manifestOf(
    sample('org.example.frontend', [EDITOR, 'desktop.flashcards']),
  ),
  'with-dist/dist/index.js': 'x',
  'flashcards-only/manifest.json': manifestOf(sample('org.example.cards', ['desktop.flashcards'])),
  'flashcards-only/dist/index.js': 'x',
}

test('a plugin with a manifest and no bin or dist is selected for its target', () => {
  withTree(fixture, (root) => {
    assert.ok(listSelectedPlugins(root, EDITOR).includes('only-contributions'))
  })
})

test('a plugin with a backend or a frontend is selected for its target', () => {
  withTree(fixture, (root) => {
    const selected = listSelectedPlugins(root, EDITOR)
    assert.ok(selected.includes('with-bin'))
    assert.ok(selected.includes('with-dist'))
  })
})

test('a plugin built for another target is not selected', () => {
  withTree(fixture, (root) => {
    assert.ok(!listSelectedPlugins(root, EDITOR).includes('flashcards-only'))
    assert.deepEqual(listSelectedPlugins(root, 'desktop.flashcards'), ['flashcards-only', 'with-dist'])
  })
})

test('the selection is ordered by folder name', () => {
  withTree(fixture, (root) => {
    assert.deepEqual(listSelectedPlugins(root, EDITOR), ['only-contributions', 'with-bin', 'with-dist'])
  })
})

test('the selection for the editor equals the plugins the index lists for it', () => {
  const byId = new Map(readPluginManifests(PLUGINS_DIR).map((one) => [one.manifest.id, one.name]))
  const expected = buildIndex(PLUGINS_DIR)
    .plugins.filter((plugin) => (plugin.targets ?? []).includes(EDITOR))
    .map((plugin) => byId.get(plugin.id))
    .sort((a, b) => a.localeCompare(b))

  assert.ok(expected.includes('themes-presets'))
  assert.deepEqual(listSelectedPlugins(PLUGINS_DIR, EDITOR), expected)
})

test('the selection for the flashcards window equals the plugins the index lists for it', () => {
  const byId = new Map(readPluginManifests(PLUGINS_DIR).map((one) => [one.manifest.id, one.name]))
  const expected = buildIndex(PLUGINS_DIR)
    .plugins.filter((plugin) => (plugin.targets ?? []).includes('desktop.flashcards'))
    .map((plugin) => byId.get(plugin.id))
    .sort((a, b) => a.localeCompare(b))

  assert.deepEqual(listSelectedPlugins(PLUGINS_DIR, 'desktop.flashcards'), expected)
})

test('the check passes when the application holds every selected plugin', () => {
  withTree(fixture, (root) => {
    const selected = listSelectedPlugins(root, EDITOR)
    assert.ok(selected.length > 0)
    const app = mkdtempSync(join(tmpdir(), 'bundle-app-'))
    try {
      for (const name of selected) {
        mkdirSync(join(app, name), { recursive: true })
        writeFileSync(join(app, name, 'manifest.json'), '{}')
      }
      const result = runSelect(['--target', EDITOR, '--plugins', root, '--verify', app])
      assert.equal(result.status, 0, result.stderr)
    } finally {
      rmSync(app, { recursive: true, force: true })
    }
  })
})

test('the check fails and names a plugin the application does not hold', () => {
  withTree(fixture, (root) => {
    const app = mkdtempSync(join(tmpdir(), 'bundle-app-'))
    try {
      for (const name of ['with-bin', 'with-dist']) {
        mkdirSync(join(app, name), { recursive: true })
        writeFileSync(join(app, name, 'manifest.json'), '{}')
      }
      const result = runSelect(['--target', EDITOR, '--plugins', root, '--verify', app])
      assert.notEqual(result.status, 0)
      assert.match(result.stdout + result.stderr, /only-contributions/)
    } finally {
      rmSync(app, { recursive: true, force: true })
    }
  })
})

test('a plugin of the index that has no folder is named in the error', () => {
  const index = [
    { id: 'org.example.known', targets: [EDITOR] },
    { id: 'org.example.orphan', targets: [EDITOR] },
  ]
  const folderById = new Map([['org.example.known', 'known']])

  assert.throws(() => listFolders(index, folderById, EDITOR), /org\.example\.orphan/)
})

test('the folders of the plugins listed for the target are ordered by name', () => {
  const index = [
    { id: 'b', targets: [EDITOR] },
    { id: 'a', targets: [EDITOR] },
    { id: 'c', targets: ['desktop.flashcards'] },
  ]
  const folderById = new Map([
    ['a', 'zeta'],
    ['b', 'alpha'],
    ['c', 'other'],
  ])

  assert.deepEqual(listFolders(index, folderById, EDITOR), ['alpha', 'zeta'])
})
