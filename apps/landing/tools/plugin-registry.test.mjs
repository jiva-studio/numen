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
import { isCardShown } from '../src/scripts/plugin-catalog.ts'

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

const card = {
  search: 'word count counts the words of a note example author',
  category: 'tools',
  apps: ['desktop.editor'],
  isPaid: false,
  isOfficial: false,
}

const noFilter = { query: '', category: '', app: '', price: '', isOfficialOnly: false }

test('a card is shown when every search word is in its text', () => {
  assert.equal(isCardShown(card, { ...noFilter, query: 'count note' }), true)
  assert.equal(isCardShown(card, { ...noFilter, query: 'count pdf' }), false)
})

test('a card is hidden by a category, an app, a price or the official filter it does not match', () => {
  assert.equal(isCardShown(card, { ...noFilter, category: 'views' }), false)
  assert.equal(isCardShown(card, { ...noFilter, app: 'desktop.flashcards' }), false)
  assert.equal(isCardShown(card, { ...noFilter, price: 'paid' }), false)
  assert.equal(isCardShown(card, { ...noFilter, price: 'free' }), true)
  assert.equal(isCardShown(card, { ...noFilter, isOfficialOnly: true }), false)
})

test('a search matches whatever the case and the spaces around it', () => {
  assert.equal(isCardShown(card, { ...noFilter, query: '  Word COUNT ' }), true)
})
