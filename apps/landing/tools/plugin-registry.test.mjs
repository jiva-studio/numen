/**
 * How the site reads the plugin registry and how the catalog filters it, from a fixture. Nothing
 * goes to the network.
 */
import assert from 'node:assert/strict'
import { existsSync, readFileSync, readdirSync } from 'node:fs'
import test from 'node:test'
import { fileURLToPath } from 'node:url'
import {
  REGISTRY_FILE_VARIABLE,
  REGISTRY_URL,
  parseRegistry,
  readRegistry,
} from '../src/components/plugins/registry.ts'
import {
  getSlotPlace,
  listContributionSections,
  listQualitySections,
} from '../src/components/plugins/sections.ts'
import { getContributorUrl } from '../src/components/plugins/catalog.ts'
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

const readPlugin = async (id) => {
  const registry = await readRegistry({ [REGISTRY_FILE_VARIABLE]: FIXTURE }, refuseFetch)
  return { ...registry.plugins.find((one) => one.id === id), changelog: [], featuredRank: null }
}

const readRows = (section) =>
  section.rows.map((row) => [
    row.name.map((one) => one.text).join(' '),
    row.value.map((line) => line.map((one) => one.text).join(' ')).join(' / '),
  ])

test('the Contributions tab shows only the sections with rows', async () => {
  const sections = listContributionSections(await readPlugin('md.numen.engines.transcription'))

  assert.deepEqual(
    sections.map((one) => one.title),
    ['Permissions', 'Settings', 'Text versions', 'Slots', 'Downloads'],
  )
  assert.deepEqual(readRows(sections[2]), [['Transcript', 'transcript']])
  assert.equal(sections[0].rows.find((row) => row.isNew)?.value[0][0].text, 'link:open')
})

test('a slot row names what the plugin puts there and the place in the window, in words', async () => {
  const plugin = await readPlugin('md.numen.engines.transcription')
  const slots = listContributionSections({
    ...plugin,
    slots: [
      { slot: 'fileTree.context', title: 'Transcribe', target: 'app:file.createText' },
      { slot: 'titleBar.status', title: '', target: 'sync' },
      { slot: 'ledger.board', title: 'Ledger', target: 'board' },
    ],
  }).find((one) => one.id === 'slots')

  assert.deepEqual(readRows(slots), [
    ['Transcribe', 'File tree menu'],
    ['sync', 'Title bar'],
    ['Ledger', 'ledger.board'],
  ])
  assert.deepEqual(slots.rows[0].value[0][0], {
    text: 'File tree menu',
    key: 'plugin_slot_fileTree_context',
    isCode: false,
  })
  assert.equal(slots.rows[1].name[0].isCode, true)
  assert.equal(slots.rows[2].value[0][0].isCode, true)
})

const ROOT = new URL('../../../', import.meta.url)

const readJson = (path) => JSON.parse(readFileSync(new URL(path, ROOT), 'utf8'))

/** The slots of the application and every slot a plugin of the repository draws or fills. */
const listKnownSlots = () => {
  const names = new Set(readJson('libs/shared/protocol/slots.json').slots.map((one) => one.name))
  for (const folder of readdirSync(new URL('plugins/', ROOT), { withFileTypes: true })) {
    const manifest = `plugins/${folder.name}/manifest.json`
    if (!existsSync(new URL(manifest, ROOT))) continue
    const contributes = readJson(manifest).contributes ?? {}
    for (const one of [...(contributes.slots ?? []), ...(contributes.slotPoints ?? [])]) {
      names.add(one.slot)
    }
  }
  return [...names]
}

test('every slot of the application and of the plugins has words in each language', async () => {
  const { EN } = await import('../src/scripts/i18n-en.ts')
  const { RU } = await import('../src/scripts/i18n-ru.ts')
  const { SR } = await import('../src/scripts/i18n-sr.ts')

  for (const slot of listKnownSlots()) {
    const place = getSlotPlace(slot)
    assert.equal(place.isCode, false, slot)
    assert.ok(RU[place.key] && SR[place.key] && EN[place.key], slot)
  }
})

test('a plugin that adds nothing has no Contributions sections', async () => {
  assert.deepEqual(listContributionSections(await readPlugin('md.numen.tools.sync')), [])
})

test('the Tests section lists the counted tests and the measured numbers, without zero counts', async () => {
  const [checks, , , tests] = listQualitySections(await readPlugin('md.numen.engines.transcription'))

  assert.equal(checks.title, 'Checks')
  assert.equal(checks.rows[0].status, 'pass')
  assert.deepEqual(readRows(tests), [
    ['Go tests', '162'],
    ['Mutation score', '78.4%'],
    ['Coverage', '71.2%'],
  ])
  assert.equal(tests.note, null)
})

test('the Quality tab shows the checks, the vulnerabilities, the dependencies and the tests', async () => {
  const sections = listQualitySections(await readPlugin('md.numen.engines.transcription'))

  assert.deepEqual(
    sections.map((one) => one.title),
    ['Checks', 'Vulnerabilities', 'Dependencies', 'Tests'],
  )
})

test('a check detail of several lines is one value line each', async () => {
  const [checks] = listQualitySections(await readPlugin('md.numen.engines.transcription'))
  const network = checks.rows.find((one) => one.name[0].key === 'plugin_check_network')

  assert.equal(network.status, 'info')
  assert.deepEqual(
    network.value.map((line) => line[0].text),
    ['Yes, read from the code and the manifest.', 'Downloads from: huggingface.co.'],
  )
})

test('a vulnerability links to its OSV page and names the package it was found in', async () => {
  const [, vulnerabilities] = listQualitySections(await readPlugin('md.numen.engines.transcription'))

  assert.deepEqual(readRows(vulnerabilities), [
    ['GO-2026-0001', 'golang.org/x/net 0.62.0 / Header parsing loops'],
  ])
  assert.equal(vulnerabilities.rows[0].href, 'https://osv.dev/vulnerability/GO-2026-0001')
})

test('the Dependencies section lists each package with its version and license, then the indirect count', async () => {
  const [, , dependencies] = listQualitySections(await readPlugin('md.numen.engines.transcription'))

  assert.deepEqual(readRows(dependencies), [
    ['github.com/yalue/onnxruntime_go', 'Go v1.21.0 MIT'],
    ['@lucide/vue', 'npm 1.72.0 license unknown'],
    ['Indirect Go modules', '4'],
  ])
  assert.equal(dependencies.rows[1].value[0][2].key, 'plugin_license_unknown')
})

test('the numbers an author reports are marked as theirs', async () => {
  const [, tests] = listQualitySections(await readPlugin('org.example.word-count'))

  assert.equal(tests.note?.key, 'plugin_quality_author')
})

test('a plugin with no quality has only its checks', async () => {
  const sections = listQualitySections(await readPlugin('md.numen.tools.sync'))

  assert.deepEqual(
    sections.map((one) => one.id),
    ['checks'],
  )
})

test('a contributor with a GitHub login links to their profile', () => {
  assert.equal(getContributorUrl({ login: 'ann-b' }), 'https://github.com/ann-b')
  assert.equal(getContributorUrl({ login: null }), null)
  assert.equal(getContributorUrl({ login: '../evil' }), null)
})
