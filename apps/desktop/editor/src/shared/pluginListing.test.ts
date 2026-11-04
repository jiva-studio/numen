import { describe, expect, it } from 'vitest'
import { createMockPluginInfo } from '@numen/plugin-manager/testing'
import { CATALOG_STATE } from '@numen/core-client'
import { createManagedFromEntries, createPluginListing } from './pluginListing'
import { createMockCatalogEntry } from '@numen/core-client/testing'

describe('createPluginListing', () => {
  it('reads the lowercase category, the feature and the tab kinds of an entry', () => {
    const listing = createPluginListing(
      createMockCatalogEntry('notes', {
        category: 'Editors',
        feature: 'Markdown notes',
        tabKinds: ['note', { kind: 'reader:markdown', title: 'Markdown' }],
      }),
    )

    expect(listing).toStrictEqual({
      id: 'notes',
      name: 'notes',
      version: '1.0.0',
      description: '',
      category: 'editors',
      feature: 'Markdown notes',
      tabKinds: ['note', 'reader:markdown'],
      isOffered: false,
    })
  })

  it('marks only an available entry as offered', () => {
    const available = createMockCatalogEntry('a', { state: CATALOG_STATE.AVAILABLE })
    const disabled = createMockCatalogEntry('b', { state: CATALOG_STATE.DISABLED })

    expect(createPluginListing(available).isOffered).toBe(true)
    expect(createPluginListing(disabled).isOffered).toBe(false)
  })

  it('falls back to top-level category and empty defaults when contributes is absent', () => {
    const base = createMockCatalogEntry('legacy')
    const withoutContributes = {
      ...base,
      category: 'Tools',
      contributes: undefined,
    }
    expect(createPluginListing(withoutContributes)).toStrictEqual({
      id: 'legacy',
      name: 'legacy',
      version: '1.0.0',
      description: '',
      category: 'tools',
      feature: '',
      tabKinds: [],
      isOffered: false,
    })
  })
})

describe('createManagedFromEntries', () => {
  it('lists the installed plugin with its entry, then the entries on offer', () => {
    const merged = createManagedFromEntries(
      [createMockPluginInfo('a', 'A')],
      [
        createMockCatalogEntry('a', { category: 'views' }),
        createMockCatalogEntry('b', { state: CATALOG_STATE.AVAILABLE }),
      ],
    )

    expect(merged.map((one) => [one.id, one.category, one.info === undefined])).toStrictEqual([
      ['a', 'views', false],
      ['b', '', true],
    ])
  })
})
