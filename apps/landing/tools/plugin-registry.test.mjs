/**
 * How the site reads the plugin registry and how the catalog filters it, from a fixture. Nothing
 * goes to the network.
 */
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'
import { fileURLToPath } from 'node:url'
import {
  REGISTRY_FILE_VARIABLE,
  REGISTRY_URL,
  parseRegistry,
  readRegistry,
} from '../src/components/plugins/registry.ts'
import { isEntryShown } from '../src/scripts/plugin-catalog.ts'

const FIXTURE = fileURLToPath(new URL('./fixtures/plugin-registry.json', import.meta.url))

const refuseFetch = async () => {
  throw new Error('the registry file was not read')
}

test('the registry is read from the file the environment names', async () => {
  const registry = await readRegistry({ [REGISTRY_FILE_VARIABLE]: FIXTURE }, refuseFetch)

  assert.deepEqual(
    registry.plugins.map((one) => one.id),
    ['md.numen.engines.transcription', 'md.numen.tools.sync', 'org.example.word-count'],
  )
})

test('the registry is fetched from its published address when no file is named', async () => {
  const requests = []
  const fetcher = async (url) => {
    requests.push(url)
    return new Response(readFileSync(FIXTURE, 'utf8'), { status: 200 })
  }
  const registry = await readRegistry({}, fetcher)

  assert.deepEqual(requests, [REGISTRY_URL])
  assert.equal(registry.plugins.length, 3)
})

test('a registry answered with an error status fails the build', async () => {
  const fetcher = async () => new Response('', { status: 404 })

  await assert.rejects(readRegistry({}, fetcher), /answered with 404/)
})

test('a registry that cannot be fetched fails the build', async () => {
  const fetcher = async () => {
    throw new TypeError('fetch failed')
  }

  await assert.rejects(readRegistry({}, fetcher), /fetch failed/)
})

test('a registry of another schema or with no plugins is rejected', () => {
  assert.throws(() => parseRegistry('{"schema":2,"plugins":[]}', 'x'), /not a plugin registry/)
  assert.throws(() => parseRegistry('{"schema":1,"plugins":[]}', 'x'), /not a plugin registry/)
})

test('a registry lists the ids it features', async () => {
  const registry = await readRegistry({ [REGISTRY_FILE_VARIABLE]: FIXTURE }, refuseFetch)

  assert.deepEqual(registry.featured, ['md.numen.engines.transcription'])
  assert.throws(
    () => parseRegistry(JSON.stringify({ ...registry, featured: undefined }), 'x'),
    /not a plugin registry/,
  )
})

const entry = { search: 'word count counts the words of a note example author', category: 'tools' }

const noFilter = { query: '', category: '' }

test('a row is shown when every search word is in its text', () => {
  assert.equal(isEntryShown(entry, { ...noFilter, query: 'count note' }), true)
  assert.equal(isEntryShown(entry, { ...noFilter, query: 'count pdf' }), false)
})

test('a row is hidden by a category it is not in', () => {
  assert.equal(isEntryShown(entry, { ...noFilter, category: 'views' }), false)
  assert.equal(isEntryShown(entry, { ...noFilter, category: 'tools' }), true)
})

test('a search matches whatever the case and the spaces around it', () => {
  assert.equal(isEntryShown(entry, { ...noFilter, query: '  Word COUNT ' }), true)
})
