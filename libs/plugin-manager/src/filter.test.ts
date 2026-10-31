import { describe, expect, it } from 'vitest'
import { CATALOG_STATE } from '@numen/plugin-sdk'
import { createMockCatalogEntry, createMockPluginInfo } from '@numen/plugin-sdk/testing'
import { createManagedPlugins } from './plugins'
import {
  formatCategoryLabel,
  getFilteredPluginItems,
  getPluginCategory,
  matchesPluginQuery,
  matchesSystemRow,
  parsePluginQuery,
  type PluginFilterContext,
} from './filter'

const createContext = (disabledIds: readonly string[] = []): PluginFilterContext => ({
  isEnabled: (id) => !disabledIds.includes(id),
})

const notesInfo = createMockPluginInfo('sample.alpha', 'Notes', {
  description: 'Markdown note editor',
  readers: [{ id: 'markdown', extensions: ['md'] }],
})
const treeInfo = createMockPluginInfo('sample.beta', 'File Tree Explorer', {
  description: 'Browse vault files',
})
const agentInfo = createMockPluginInfo('sample.gamma', 'Claude Code Agent', {
  capabilities: ['agent', 'ai.agent.provider'],
})
const bareInfo = createMockPluginInfo('sample.delta', 'Bare Engine')

const entries = [
  createMockCatalogEntry('sample.alpha', {
    category: 'editors',
    feature: 'Markdown notes',
    tabKinds: ['note', 'reader:markdown'],
  }),
  createMockCatalogEntry('sample.beta', {
    category: 'views',
    feature: 'file tree',
    tabKinds: ['files'],
  }),
  createMockCatalogEntry('sample.gamma', { category: 'agents' }),
  createMockCatalogEntry('sample.offer', {
    name: 'Offered Reader',
    category: 'readers',
    feature: 'EPUB books',
    tabKinds: ['reader:offer'],
    state: CATALOG_STATE.AVAILABLE,
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
    expect([notes.category, notes.feature, notes.tabKinds]).toStrictEqual([
      'editors',
      'Markdown notes',
      ['note', 'reader:markdown'],
    ])
  })

  it('leaves a plugin without an entry in no category', () => {
    expect([bare.category, bare.feature, bare.tabKinds]).toStrictEqual(['', '', []])
    expect(getPluginCategory(bare)).toBe('other')
  })

  it('appends the entries the catalog offers that are not installed', () => {
    expect([uninstalled.id, uninstalled.info, uninstalled.name]).toStrictEqual([
      'sample.offer',
      undefined,
      'Offered Reader',
    ])
  })

  it('does not list an uninstalled entry of a plugin that is installed', () => {
    const merged = createManagedPlugins(
      [notesInfo],
      [createMockCatalogEntry('sample.alpha', { state: CATALOG_STATE.AVAILABLE })],
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
    expect(parsePluginQuery('@available').qualifiers).toStrictEqual([
      { key: 'status', value: 'available' },
    ])
  })
})

describe('the feature: filter', () => {
  it('matches the feature the manifest declares, not the id or the name', () => {
    const ctx = createContext()

    expect(matchesPluginQuery(tree, parsePluginQuery('feature:file'), ctx)).toBe(true)
    expect(matchesPluginQuery(tree, parsePluginQuery('feature:explorer'), ctx)).toBe(false)
    expect(matchesPluginQuery(tree, parsePluginQuery('feature:beta'), ctx)).toBe(false)
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
    expect(matchesPluginQuery(agent, parsePluginQuery('category:AGENTS'), createContext())).toBe(
      true,
    )
  })
})

describe('matchesPluginQuery and matchesSystemRow', () => {
  it('combines multiple qualifiers and text terms', () => {
    const ctx = createContext(['sample.alpha'])
    const query = parsePluginQuery('@disabled reader:markdown note')

    expect(matchesPluginQuery(notes, query, ctx)).toBe(true)
    expect(matchesPluginQuery(tree, query, ctx)).toBe(false)
    expect(matchesSystemRow(query)).toBe(false)
  })

  it('matches capability and id qualifiers', () => {
    const ctx = createContext()

    expect(matchesPluginQuery(agent, parsePluginQuery('capability:agent'), ctx)).toBe(true)
    expect(matchesPluginQuery(notes, parsePluginQuery('id:sample.alpha'), ctx)).toBe(true)
  })

  it('matches readers by id and extension only', () => {
    const ctx = createContext()

    expect(matchesPluginQuery(notes, parsePluginQuery('reader:md'), ctx)).toBe(true)
    expect(matchesPluginQuery(notes, parsePluginQuery('reader:alpha'), ctx)).toBe(false)
    expect(matchesPluginQuery(notes, parsePluginQuery('reader:'), ctx)).toBe(true)
    expect(matchesPluginQuery(tree, parsePluginQuery('reader:'), ctx)).toBe(false)
  })

  it('applies the available status to plugins that are not installed only', () => {
    const ctx = createContext(['sample.alpha'])

    expect(matchesPluginQuery(uninstalled, parsePluginQuery('@available'), ctx)).toBe(true)
    expect(matchesPluginQuery(notes, parsePluginQuery('@available'), ctx)).toBe(false)
    expect(matchesPluginQuery(uninstalled, parsePluginQuery('@disabled'), ctx)).toBe(false)
    expect(matchesPluginQuery(uninstalled, parsePluginQuery('@enabled'), ctx)).toBe(false)
  })

  it('finds free text in the feature of a plugin', () => {
    expect(matchesPluginQuery(uninstalled, parsePluginQuery('epub'), createContext())).toBe(true)
  })

  it('matches the system row only without plugin qualifiers and with matching keywords', () => {
    expect(matchesSystemRow(parsePluginQuery(''))).toBe(true)
    expect(matchesSystemRow(parsePluginQuery('theme'))).toBe(true)
    expect(matchesSystemRow(parsePluginQuery('@disabled'))).toBe(false)
    expect(matchesSystemRow(parsePluginQuery('whisper'))).toBe(false)
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

  it('keeps only the plugins that match the query', () => {
    const items = getFilteredPluginItems(
      [tree, notes, agent],
      parsePluginQuery('@disabled'),
      createContext(['sample.alpha']),
    )

    expect(items.map((item) => item.plugin.id)).toStrictEqual(['sample.alpha'])
  })

  it('formats the label of a category', () => {
    expect(formatCategoryLabel('views')).toBe('Views')
  })
})
