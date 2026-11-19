import { effectScope } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { createMockHostServices, createMockPluginContext } from '@numen/plugin-sdk/testing'
import { activate, LOGS_KIND, OPEN_LOGS_COMMAND } from './index'
import type { LogsView } from './model/logs'
import LogsTab from './ui/LogsTab.vue'

describe('the Logs plugin', () => {
  it('registers the logs tab kind and opens it on the logs.open command', async () => {
    const opened: string[] = []
    const plugin = createMockPluginContext({ pluginId: 'md.numen.tools.logs' })
    const host = createMockHostServices({
      openTab: (kind) => {
        opened.push(kind)
        return Promise.resolve(`tab:${kind}`)
      },
    })

    activate(plugin, host)

    expect(plugin.getRegisteredTabKinds().map((one) => one.kind)).toStrictEqual([LOGS_KIND])
    expect(plugin.getRegisteredTabKinds()[0]?.icon).toBeDefined()
    await plugin.triggerCommand(OPEN_LOGS_COMMAND)
    expect(opened).toStrictEqual([LOGS_KIND])
  })

  it('renders log rows, filters by level and search query, and expands SQL and JSON details on click', async () => {
    const plugin = createMockPluginContext({ pluginId: 'md.numen.tools.logs' })
    const host = createMockHostServices()
    activate(plugin, host)

    const kind = plugin.getRegisteredTabKinds()[0]
    if (!kind?.open) throw new Error('logs tab kind did not declare open')

    const scope = effectScope()
    const state = scope.run(() => kind.open?.('', 'tab-1')) as LogsView | undefined
    if (!state) throw new Error('open returned no state')

    host.work.logs.pushEntries([
      {
        time: Date.UTC(2026, 10, 18, 12, 0, 0, 0),
        level: 'debug',
        source: 'sql',
        message: 'SQL query',
        properties: {
          pool: '"read"',
          sql: '"SELECT id, path FROM notes WHERE vault_id = ? ORDER BY path"',
          durationMs: '0.84',
        },
      },
      {
        time: Date.UTC(2026, 10, 18, 12, 0, 1, 0),
        level: 'info',
        source: 'rpc',
        message: 'NoteService/CreateNote',
        properties: {
          request: '{"vaultId":"v1","title":"Welcome"}',
          durationMs: '3.2',
        },
      },
      {
        time: Date.UTC(2026, 10, 18, 12, 0, 2, 0),
        level: 'error',
        source: 'md.numen.engines.ocr',
        message: 'Recognition failed',
        properties: {
          error: '"missing model"',
        },
      },
    ])

    const wrapper = mount(LogsTab, { props: { state } })

    host.work.logs.pushEntries([
      {
        time: Date.UTC(2026, 10, 18, 12, 0, 3, 0),
        level: 'info',
        source: 'window',
        message: 'Opened logs',
        properties: { tab: '"logs"' },
      },
    ])
    await wrapper.vm.$nextTick()

    const rows = wrapper.findAll('.log-row')
    expect(rows).toHaveLength(4)
    expect(rows[0]?.text()).toContain('Opened logs')
    await rows[0]?.find('.log-row__summary').trigger('click')
    expect(rows[0]?.find('.log-row__details').exists()).toBe(true)
    expect(rows[1]?.find('.log-row__dot--error').exists()).toBe(true)
    expect(rows[3]?.text()).toContain('SQL query')

    // Expand the SQL row and verify formatted and syntax-highlighted SQL block
    await rows[3]?.find('.log-row__summary').trigger('click')
    expect(rows[3]?.find('.log-row__code').text()).toContain(
      'SELECT id, path\nFROM notes\nWHERE vault_id = ?\nORDER BY path',
    )
    expect(rows[3]?.find('.log-row__token--keyword').text()).toBe('SELECT')

    // Expand the RPC row and verify formatted and syntax-highlighted JSON block
    await rows[2]?.find('.log-row__summary').trigger('click')
    expect(rows[2]?.find('.log-row__code').text()).toContain(
      '{\n  "vaultId": "v1",\n  "title": "Welcome"\n}',
    )
    expect(rows[2]?.find('.log-row__token--key').text()).toBe('"vaultId"')

    const levelGroups = wrapper.findAll('.logs-tab__levels')
    await levelGroups[1]?.findAll('button')[0]?.trigger('click')
    expect(state.logLevel.value).toBe('debug')

    // Filter by Error level
    const errorBtn = wrapper.findAll('.logs-tab__level-btn').find((btn) => btn.text() === 'Error')
    await errorBtn?.trigger('click')
    expect(wrapper.findAll('.log-row')).toHaveLength(1)
    expect(wrapper.find('.log-row__message').text()).toBe('Recognition failed')

    // Switch back to All and filter by search query
    const allBtn = wrapper.findAll('.logs-tab__level-btn').find((btn) => btn.text() === 'All')
    await allBtn?.trigger('click')
    await wrapper.find('.logs-tab__search').setValue('CreateNote')
    expect(wrapper.findAll('.log-row')).toHaveLength(1)
    expect(wrapper.find('.log-row__message').text()).toBe('NoteService/CreateNote')

    scope.stop()
    expect(host.work.logs.countWatchers()).toBe(0)
  })
})
