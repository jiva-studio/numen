import { describe, expect, it } from 'vitest'
import { createPluginInfo } from '@/testing/installation'
import {
  formatCategoryLabel,
  getFilteredPluginItems,
  getPluginCategory,
  matchesPluginQuery,
  matchesSystemRow,
  parsePluginQuery,
  type PluginFilterContext,
} from './pluginFilter'

const createContext = (disabledIds: readonly string[] = []): PluginFilterContext => ({
  isEnabled: (id) => !disabledIds.includes(id),
  getCategory: getPluginCategory,
  getCategoryLabel: (cat) => formatCategoryLabel(cat, { views: 'Views', readers: 'Readers' }),
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
  })
})

describe('matchesPluginQuery and matchesSystemRow', () => {
  const notesPlugin = createPluginInfo('md.numen.notes', 'Notes', {
    description: 'Markdown note editor',
    readers: [{ id: 'markdown', extensions: ['md'] }],
  })
  const filesPlugin = createPluginInfo('md.numen.views.file.tree', 'File Tree Explorer', {
    description: 'Browse vault files',
  })
  const agentPlugin = createPluginInfo('md.numen.agents.claude', 'Claude Code Agent', {
    capabilities: ['agent', 'ai.agent.provider'],
  })

  it('combines multiple qualifiers and text terms', () => {
    const ctx = createContext(['md.numen.notes'])
    const query = parsePluginQuery('@disabled reader:markdown note')

    expect(matchesPluginQuery(notesPlugin, query, ctx)).toBe(true)
    expect(matchesPluginQuery(filesPlugin, query, ctx)).toBe(false)
    expect(matchesSystemRow(query)).toBe(false)
  })

  it('matches feature, category, capability, and id qualifiers', () => {
    const ctx = createContext()

    expect(matchesPluginQuery(filesPlugin, parsePluginQuery('feature:files'), ctx)).toBe(true)
    expect(matchesPluginQuery(filesPlugin, parsePluginQuery('category:views'), ctx)).toBe(true)
    expect(matchesPluginQuery(agentPlugin, parsePluginQuery('capability:agent'), ctx)).toBe(true)
    expect(matchesPluginQuery(notesPlugin, parsePluginQuery('id:md.numen.notes'), ctx)).toBe(true)
  })

  it('matches the system row only without plugin qualifiers and with matching keywords', () => {
    expect(matchesSystemRow(parsePluginQuery(''))).toBe(true)
    expect(matchesSystemRow(parsePluginQuery('theme'))).toBe(true)
    expect(matchesSystemRow(parsePluginQuery('@disabled'))).toBe(false)
    expect(matchesSystemRow(parsePluginQuery('whisper'))).toBe(false)
  })

  it('filters, categorizes, and sorts plugins by category order and name', () => {
    const ctx = createContext(['md.numen.notes'])
    const items = getFilteredPluginItems(
      [filesPlugin, notesPlugin, agentPlugin],
      parsePluginQuery(''),
      ctx,
    )
    expect(items.map((item) => [item.category, item.plugin.name])).toStrictEqual([
      ['agents', 'Claude Code Agent'],
      ['views', 'File Tree Explorer'],
      ['notes', 'Notes'],
    ])

    const disabledItems = getFilteredPluginItems(
      [filesPlugin, notesPlugin, agentPlugin],
      parsePluginQuery('@disabled'),
      ctx,
    )
    expect(disabledItems.map((item) => item.plugin.id)).toStrictEqual(['md.numen.notes'])
  })
})
