import { describe, expect, it } from 'vitest'
import { create } from '@bufbuild/protobuf'
import {
  FindCatalogEntryByTabKindResponseSchema,
  GetCatalogEntryResponseSchema,
  ListCatalogResponseSchema,
} from '@numen/protocol'
import {
  CATALOG_ERROR_CODE,
  CATALOG_STATE,
  createPluginCatalog,
  type CatalogClient,
} from './catalog'
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
    const askedKinds: Array<string | undefined> = []
    const client = createClient({
      findCatalogEntryByTabKind: (req) => {
        askedKinds.push(req.kind)
        return Promise.resolve(create(FindCatalogEntryByTabKindResponseSchema, { entry: notes }))
      },
    })
    const answer = await createPluginCatalog(client).findByTabKind('note')

    expect(askedKinds).toStrictEqual(['note'])
    expect(answer.ok).toBe(true)
    if (!answer.ok) return
    expect(answer.value.entry.contributes?.feature).toBe('Markdown notes')
  })

  it('gets the entry for a requested plugin id', async () => {
    const askedIds: Array<string | undefined> = []
    const client = createClient({
      getCatalogEntry: (req) => {
        askedIds.push(req.id)
        return Promise.resolve(create(GetCatalogEntryResponseSchema, { entry: notes }))
      },
    })
    const answer = await createPluginCatalog(client).get('notes')

    expect(askedIds).toStrictEqual(['notes'])
    expect(answer.ok).toBe(true)
  })

  it('reports not-found when the catalog has no entry for the kind', async () => {
    const client = createClient({
      findCatalogEntryByTabKind: () =>
        Promise.resolve(
          create(FindCatalogEntryByTabKindResponseSchema, {
            error: CATALOG_ERROR_CODE.NOT_FOUND,
            entry: notes,
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
          create(GetCatalogEntryResponseSchema, { error: CATALOG_ERROR_CODE.UNSPECIFIED }),
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
