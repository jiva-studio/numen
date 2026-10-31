import { describe, expect, it } from 'vitest'
import { create } from '@bufbuild/protobuf'
import {
  CatalogErrorCode,
  FindCatalogEntryByTabKindResponseSchema,
  GetCatalogEntryResponseSchema,
  ListCatalogResponseSchema,
} from '@numen/protocol'
import { CATALOG_STATE, createPluginCatalog, parseManifest, type CatalogClient } from './catalog'
import { createMockCatalogEntry, createMockPluginCatalog } from './testing/mockCatalog'

const notes = createMockCatalogEntry('notes', { tabKinds: ['note'], feature: 'Markdown notes' })

const createClient = (overrides: Partial<CatalogClient> = {}): CatalogClient => ({
  listCatalog: () =>
    Promise.resolve(
      create(ListCatalogResponseSchema, {
        entries: [notes],
        sourceErrors: [{ source: 'remote', message: 'offline' }],
      }),
    ),
  getCatalogEntry: () => Promise.resolve(create(GetCatalogEntryResponseSchema, { entry: notes })),
  findCatalogEntryByTabKind: () =>
    Promise.resolve(create(FindCatalogEntryByTabKindResponseSchema, { entry: notes })),
  ...overrides,
})

describe('createPluginCatalog', () => {
  it('lists the entries with the sources that failed', async () => {
    const answer = await createPluginCatalog(createClient()).list()

    expect(answer.ok).toBe(true)
    if (!answer.ok) return
    expect(answer.value.entries.map((one) => one.id)).toStrictEqual(['notes'])
    expect(answer.value.sourceErrors).toHaveLength(1)
  })

  it('returns the entry that provides a kind of tab', async () => {
    const answer = await createPluginCatalog(createClient()).findByTabKind('note')

    expect(answer.ok).toBe(true)
    if (!answer.ok) return
    expect(answer.value.entry.contributes?.feature).toBe('Markdown notes')
  })

  it('reports not-found when the catalog has no entry for the kind', async () => {
    const client = createClient({
      findCatalogEntryByTabKind: () =>
        Promise.resolve(
          create(FindCatalogEntryByTabKindResponseSchema, {
            error: CatalogErrorCode.NOT_FOUND,
          }),
        ),
    })

    const answer = await createPluginCatalog(client).findByTabKind('missing')

    expect(answer).toStrictEqual({ ok: false, error: { kind: 'not-found', sourceErrors: [] } })
  })

  it('reports not-found for an id no source lists', async () => {
    const client = createClient({
      getCatalogEntry: () =>
        Promise.resolve(
          create(GetCatalogEntryResponseSchema, { error: CatalogErrorCode.NOT_FOUND }),
        ),
    })

    const answer = await createPluginCatalog(client).get('missing')

    expect(answer.ok).toBe(false)
  })

  it('reports unavailable when a call fails', async () => {
    const client = createClient({
      listCatalog: () => Promise.reject(new Error('connection lost')),
      getCatalogEntry: () => Promise.reject(new Error('connection lost')),
      findCatalogEntryByTabKind: () => Promise.reject(new Error('connection lost')),
    })
    const catalog = createPluginCatalog(client)

    for (const answer of [
      await catalog.list(),
      await catalog.get('notes'),
      await catalog.findByTabKind('note'),
    ]) {
      expect(answer.ok).toBe(false)
      if (!answer.ok) expect(answer.error.kind).toBe('unavailable')
    }
  })
})

describe('parseManifest', () => {
  it('reads a manifest and ignores the keys the schema does not name', () => {
    const answer = parseManifest({
      id: 'sample',
      name: 'Sample',
      version: '1.0.0',
      futureField: true,
      contributes: { feature: 'Samples', tabKinds: [{ kind: 'sample' }] },
    })

    expect(answer.ok).toBe(true)
    if (!answer.ok) return
    expect(answer.value.contributes?.tabKinds[0]?.kind).toBe('sample')
  })

  it('reports a value of the wrong type', () => {
    const answer = parseManifest({ id: 5 })

    expect(answer.ok).toBe(false)
  })
})

describe('createMockPluginCatalog', () => {
  it('finds the plugin that provides a kind and records the question', async () => {
    const catalog = createMockPluginCatalog([
      createMockCatalogEntry('off', { state: CATALOG_STATE.DISABLED, tabKinds: ['off-kind'] }),
    ])

    const found = await catalog.findByTabKind('off-kind')
    const missing = await catalog.findByTabKind('other')

    expect(found.ok && found.value.entry.id).toBe('off')
    expect(missing.ok).toBe(false)
    expect(catalog.getAskedKinds()).toStrictEqual(['off-kind', 'other'])
  })

  it('gets an entry by id and follows the entries set by a test', async () => {
    const catalog = createMockPluginCatalog()

    expect((await catalog.get('a')).ok).toBe(false)
    catalog.setEntries([createMockCatalogEntry('a')])

    expect((await catalog.get('a')).ok).toBe(true)
    const listing = await catalog.list()
    expect(listing.ok && listing.value.entries).toHaveLength(1)
  })
})
