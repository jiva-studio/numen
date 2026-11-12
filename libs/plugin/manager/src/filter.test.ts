import { describe, expect, it } from 'vitest'
import { createMockListing, createMockPluginInfo } from './testing/listing'
import { createManagedPlugins, isInstalled } from './plugins'
import {
  formatCategoryLabel,
  getFilteredPluginItems,
  getPluginCategory,
  matchesPluginQuery,
  parsePluginQuery,
  type PluginFilterContext,
} from './filter'

const createContext = (disabledIds: readonly string[] = []): PluginFilterContext => ({
  isEnabled: (id) => !disabledIds.includes(id),
})

const notesInfo = createMockPluginInfo('sample.alpha', 'Notes', {
  description: 'Markdown note editor',
  readers: [{ id: 'markdown', extensions: ['txt', 'md'] }],
})
const treeInfo = createMockPluginInfo('sample.beta', 'File Tree Explorer', {
  description: 'Browse vault files',
})
const agentInfo = createMockPluginInfo('sample.gamma', 'Claude Code Agent', {
  capabilities: ['ocr', 'agent'],
})
const bareInfo = createMockPluginInfo('Sample.Delta', 'Bare Engine')

const entries = [
  createMockListing('sample.alpha', {
    category: 'editors',
    feature: 'Markdown notes',
    tabKinds: ['note', 'reader:markdown'],
  }),
  createMockListing('sample.beta', {
    category: 'views',
    feature: 'file tree',
    tabKinds: ['files'],
  }),
  createMockListing('sample.gamma', { category: 'agents' }),
  createMockListing('sample.offer', {
    name: 'Offered Reader',
    category: 'readers',
    feature: 'EPUB books',
    tabKinds: ['reader:offer'],
    isAvailable: true,
  }),
]

const [notes, tree, agent, bare, uninstalled] = createManagedPlugins(
  [notesInfo, treeInfo, agentInfo, bareInfo],
  entries,
)
if (!notes || !tree || !agent || !bare || !uninstalled)
  throw new Error('the fixtures did not merge')

describe('createManagedPlugins', () => {
  it('reads category and feature from the catalog entry of each plugin', () => {
    const [mergedNotes] = createManagedPlugins([notesInfo], entries)

    expect(mergedNotes).toBeDefined()
    if (!mergedNotes) return
    expect(isInstalled(mergedNotes)).toBe(true)
    expect([mergedNotes.category, mergedNotes.feature, mergedNotes.tabKinds]).toStrictEqual([
      'editors',
      'Markdown notes',
      ['note', 'reader:markdown'],
    ])
  })

  it('leaves a plugin without an entry in no category', () => {
    const [mergedBare] = createManagedPlugins([bareInfo], entries)

    expect(mergedBare).toBeDefined()
    if (!mergedBare) return
    expect([mergedBare.category, mergedBare.feature, mergedBare.tabKinds]).toStrictEqual([
      '',
      '',
      [],
    ])
    expect(getPluginCategory(mergedBare)).toBe('other')
  })

  it('appends the entries the catalog offers that are not installed', () => {
    const [, mergedAvailable] = createManagedPlugins([notesInfo], entries)

    expect(mergedAvailable).toBeDefined()
    if (!mergedAvailable) return
    expect(isInstalled(mergedAvailable)).toBe(false)
    expect([
      mergedAvailable.id,
      mergedAvailable.info,
      mergedAvailable.name,
      mergedAvailable.category,
      mergedAvailable.feature,
      mergedAvailable.tabKinds,
    ]).toStrictEqual([
      'sample.offer',
      undefined,
      'Offered Reader',
      'readers',
      'EPUB books',
      ['reader:offer'],
    ])
  })

  it('does not list an uninstalled entry of a plugin that is installed', () => {
    const merged = createManagedPlugins(
      [notesInfo],
      [createMockListing('sample.alpha', { isAvailable: true })],
    )

    expect(merged.map((one) => one.id)).toStrictEqual(['sample.alpha'])
  })
})

describe('parsePluginQuery', () => {
  it('parses @tag, key:value qualifiers, quoted values, and free-text terms', () => {
    expect(
      parsePluginQuery('@disabled reader:markdown @category:"views" note editor'),
    ).toStrictEqual({
      qualifiers: [
        { key: 'status', value: 'disabled' },
        { key: 'reader', value: 'markdown' },
        { key: 'category', value: 'views' },
      ],
      terms: ['note', 'editor'],
    })
  })

  it('returns empty qualifiers and terms for an empty or whitespace query', () => {
    expect(parsePluginQuery('')).toStrictEqual({ qualifiers: [], terms: [] })
    expect(parsePluginQuery('   ')).toStrictEqual({ qualifiers: [], terms: [] })
  })

  it('keeps unknown keys and leading colons as free-text terms and preserves inner quotes', () => {
    expect(parsePluginQuery(':colon idx unknown:val feature:"a"b"')).toStrictEqual({
      qualifiers: [{ key: 'feature', value: 'a"b' }],
      terms: [':colon', 'idx', 'unknown:val'],
    })
  })

  it('normalizes disabled:true/false and enabled:true/false into status qualifiers', () => {
    expect(parsePluginQuery('disabled:true').qualifiers).toStrictEqual([
      { key: 'status', value: 'disabled' },
    ])
    expect(parsePluginQuery('disabled:false').qualifiers).toStrictEqual([
      { key: 'status', value: 'enabled' },
    ])
    expect(parsePluginQuery('enabled:true').qualifiers).toStrictEqual([
      { key: 'status', value: 'enabled' },
    ])
    expect(parsePluginQuery('enabled:false').qualifiers).toStrictEqual([
      { key: 'status', value: 'disabled' },
    ])
    expect(parsePluginQuery('@available').qualifiers).toStrictEqual([
      { key: 'status', value: 'available' },
    ])
  })
})

describe('the feature: filter', () => {
  it('matches the feature the manifest declares, not the id or the name', () => {
    const ctx = createContext()

    expect(matchesPluginQuery(tree, parsePluginQuery('feature:file'), ctx)).toBe(true)
    expect(matchesPluginQuery(tree, parsePluginQuery('feature:trees'), ctx)).toBe(true)
    expect(matchesPluginQuery(notes, parsePluginQuery('feature:trees'), ctx)).toBe(false)
    expect(matchesPluginQuery(tree, parsePluginQuery('feature:" file "'), ctx)).toBe(true)
    expect(matchesPluginQuery(tree, parsePluginQuery('feature:"   "'), ctx)).toBe(false)
    expect(matchesPluginQuery(tree, parsePluginQuery('feature:explorer'), ctx)).toBe(false)
    expect(matchesPluginQuery(tree, parsePluginQuery('feature:beta'), ctx)).toBe(false)
  })

  it('does not stem short words ending in s or words not ending in s', () => {
    const ctx = createContext()
    const [shortPlugin] = createManagedPlugins(
      [createMockPluginInfo('sample.short', 'Short')],
      [createMockListing('sample.short', { feature: 'ab not' })],
    )
    expect(shortPlugin).toBeDefined()
    if (!shortPlugin) return

    expect(matchesPluginQuery(shortPlugin, parsePluginQuery('feature:abs'), ctx)).toBe(false)
    expect(matchesPluginQuery(shortPlugin, parsePluginQuery('feature:note'), ctx)).toBe(false)
  })

  it('matches a kind of tab the plugin registers', () => {
    const ctx = createContext()

    expect(matchesPluginQuery(notes, parsePluginQuery('feature:reader:markdown'), ctx)).toBe(true)
    expect(matchesPluginQuery(tree, parsePluginQuery('feature:note'), ctx)).toBe(false)
  })

  it('without a value matches the plugins that declare a feature', () => {
    const ctx = createContext()

    expect(matchesPluginQuery(notes, parsePluginQuery('feature:'), ctx)).toBe(true)
    expect(matchesPluginQuery(bare, parsePluginQuery('feature:'), ctx)).toBe(false)
  })
})

describe('the category: filter', () => {
  it('matches the category the manifest declares, not a part of the id', () => {
    const ctx = createContext()

    expect(matchesPluginQuery(tree, parsePluginQuery('category:views'), ctx)).toBe(true)
    expect(matchesPluginQuery(tree, parsePluginQuery('category:sample'), ctx)).toBe(false)
    expect(matchesPluginQuery(bare, parsePluginQuery('category:other'), ctx)).toBe(true)
  })

  it('matches the label of the category', () => {
    const ctx = createContext()
    const mixedCase = { ...bare, category: 'CustomCat' }

    expect(matchesPluginQuery(agent, parsePluginQuery('category:AGENTS'), ctx)).toBe(true)
    expect(matchesPluginQuery(mixedCase, parsePluginQuery('category:customcat'), ctx)).toBe(true)
    expect(
      matchesPluginQuery(
        mixedCase,
        { qualifiers: [{ key: 'category', value: 'CustomCat' }], terms: [] },
        ctx,
      ),
    ).toBe(true)
    expect(
      matchesPluginQuery(
        mixedCase,
        { qualifiers: [{ key: 'unknown', value: 'anything' }], terms: [] },
        ctx,
      ),
    ).toBe(true)
  })
})

describe('matchesPluginQuery', () => {
  it('combines multiple qualifiers and text terms', () => {
    const ctx = createContext(['sample.alpha'])
    const query = parsePluginQuery('@disabled reader:markdown note')

    expect(matchesPluginQuery(notes, query, ctx)).toBe(true)
    expect(matchesPluginQuery(tree, query, ctx)).toBe(false)
  })

  it('matches capability and id qualifiers including plural stems and empty value', () => {
    const ctx = createContext()

    expect(matchesPluginQuery(agent, parsePluginQuery('capability:agent'), ctx)).toBe(true)
    expect(matchesPluginQuery(agent, parsePluginQuery('capability:agents'), ctx)).toBe(true)
    expect(matchesPluginQuery(agent, parsePluginQuery('capability:'), ctx)).toBe(true)
    expect(matchesPluginQuery(bare, parsePluginQuery('capability:'), ctx)).toBe(false)
    expect(matchesPluginQuery(uninstalled, parsePluginQuery('capability:'), ctx)).toBe(false)
    expect(matchesPluginQuery(notes, parsePluginQuery('id:sample.alpha'), ctx)).toBe(true)
    expect(matchesPluginQuery(bare, parsePluginQuery('id:sample.delta'), ctx)).toBe(true)
  })

  it('matches readers by id and extension only, including plural stems', () => {
    const ctx = createContext()

    expect(matchesPluginQuery(notes, parsePluginQuery('reader:md'), ctx)).toBe(true)
    expect(matchesPluginQuery(notes, parsePluginQuery('reader:txts'), ctx)).toBe(true)
    expect(matchesPluginQuery(notes, parsePluginQuery('reader:markdowns'), ctx)).toBe(true)
    expect(matchesPluginQuery(notes, parsePluginQuery('reader:alpha'), ctx)).toBe(false)
    expect(matchesPluginQuery(notes, parsePluginQuery('reader:'), ctx)).toBe(true)
    expect(matchesPluginQuery(tree, parsePluginQuery('reader:'), ctx)).toBe(false)
  })

  it('applies status qualifiers to installed and uninstalled plugins', () => {
    const ctx = createContext(['sample.alpha'])

    expect(matchesPluginQuery(uninstalled, parsePluginQuery('@available'), ctx)).toBe(true)
    expect(matchesPluginQuery(notes, parsePluginQuery('@available'), ctx)).toBe(false)
    expect(matchesPluginQuery(uninstalled, parsePluginQuery('@disabled'), ctx)).toBe(false)
    expect(matchesPluginQuery(uninstalled, parsePluginQuery('@enabled'), ctx)).toBe(false)
    expect(matchesPluginQuery(notes, parsePluginQuery('@disabled'), ctx)).toBe(true)
    expect(matchesPluginQuery(notes, parsePluginQuery('@enabled'), ctx)).toBe(false)
    expect(matchesPluginQuery(tree, parsePluginQuery('@enabled'), ctx)).toBe(true)
    expect(matchesPluginQuery(tree, parsePluginQuery('@disabled'), ctx)).toBe(false)
  })

  it('finds free text across name, description, id, category, feature, capability, and reader', () => {
    const ctx = createContext()

    expect(matchesPluginQuery(bare, parsePluginQuery('bare'), ctx)).toBe(true)
    expect(matchesPluginQuery(tree, parsePluginQuery('browse'), ctx)).toBe(true)
    expect(matchesPluginQuery(bare, parsePluginQuery('sample.delta'), ctx)).toBe(true)
    expect(matchesPluginQuery(tree, parsePluginQuery('views'), ctx)).toBe(true)
    expect(matchesPluginQuery(uninstalled, parsePluginQuery('epub'), ctx)).toBe(true)
    expect(matchesPluginQuery(agent, parsePluginQuery('ocr'), ctx)).toBe(true)
    expect(matchesPluginQuery(notes, parsePluginQuery('txt'), ctx)).toBe(true)
    expect(matchesPluginQuery(bare, parsePluginQuery('nonexistent'), ctx)).toBe(false)
  })
})

describe('getFilteredPluginItems', () => {
  it('orders categories by label with the catch-all last, then plugins by name', () => {
    const items = getFilteredPluginItems(
      [bare, tree, notes, agent, uninstalled],
      parsePluginQuery(''),
      createContext(),
    )

    expect(items.map((item) => [item.category, item.plugin.name])).toStrictEqual([
      ['agents', 'Claude Code Agent'],
      ['editors', 'Notes'],
      ['readers', 'Offered Reader'],
      ['views', 'File Tree Explorer'],
      ['other', 'Bare Engine'],
    ])
  })

  it('keeps only the plugins that match a qualifier or free-text query', () => {
    const byQualifier = getFilteredPluginItems(
      [tree, notes, agent],
      parsePluginQuery('@disabled'),
      createContext(['sample.alpha']),
    )
    const byTerm = getFilteredPluginItems(
      [tree, notes, agent],
      parsePluginQuery('browse'),
      createContext(),
    )

    expect(byQualifier.map((item) => item.plugin.id)).toStrictEqual(['sample.alpha'])
    expect(byTerm.map((item) => item.plugin.id)).toStrictEqual(['sample.beta'])
  })

  it('formats the label of a category', () => {
    expect(formatCategoryLabel('views')).toBe('Views')
  })
})
