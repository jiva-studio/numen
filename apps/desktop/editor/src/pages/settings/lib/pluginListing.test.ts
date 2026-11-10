import { describe, expect, it } from 'vitest'
import { createMockPluginInfo } from '@numen/plugin-manager/testing'
import { createCatalogEntry } from '@/testing/installation'
import { createManagedFromEntries, createPluginListing } from './pluginListing'

describe('createPluginListing', () => {
  it('reads the lowercase category, the feature and the tab kinds of an entry', () => {
    const listing = createPluginListing(
      createCatalogEntry('notes', {
        category: 'Editors',
        feature: 'Markdown notes',
        tabKinds: ['note', 'reader:markdown'],
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
      isAvailable: false,
    })
  })

  it('marks only an available entry as offered', () => {
    const entry = createCatalogEntry('a', { isAvailable: true })
    expect(createPluginListing(entry).isAvailable).toBe(true)
    expect(createPluginListing(createCatalogEntry('b')).isAvailable).toBe(false)
  })
})

describe('createManagedFromEntries', () => {
  it('lists the installed plugin with its entry, then the entries on offer', () => {
    const merged = createManagedFromEntries(
      [createMockPluginInfo('a', 'A')],
      [
        createCatalogEntry('a', { category: 'views' }),
        createCatalogEntry('b', { isAvailable: true }),
      ],
    )

    expect(merged.map((one) => [one.id, one.category, one.info === undefined])).toStrictEqual([
      ['a', 'views', false],
      ['b', '', true],
    ])
  })
})
