import { describe, expect, it } from 'vitest'
import { createMockPluginInfo } from '@numen/plugin-manager/testing'
import { createCatalogEntry } from '@/pages/settings/testing/installation'
import { createManagedFromEntries } from './pluginListing'

describe('the listing of a catalog entry', () => {
  it('reads the lowercase category, the feature and the tab kinds of an entry', () => {
    const [result] = createManagedFromEntries(
      [createMockPluginInfo('notes', 'notes')],
      [
        createCatalogEntry('notes', {
          category: 'Editors',
          feature: 'Markdown notes',
          tabKinds: ['note', 'reader:markdown'],
        }),
      ],
    )

    expect(result).toMatchObject({
      id: 'notes',
      category: 'editors',
      feature: 'Markdown notes',
      tabKinds: ['note', 'reader:markdown'],
    })
  })

  it('marks only an available entry as available', () => {
    const available = createManagedFromEntries(
      [],
      [createCatalogEntry('a', { isAvailable: true }), createCatalogEntry('b')],
    )

    expect(available.map((one) => one.id)).toStrictEqual(['a'])
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
