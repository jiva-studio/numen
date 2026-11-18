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
    await plugin.triggerCommand(OPEN_LOGS_COMMAND)
    expect(opened).toStrictEqual([LOGS_KIND])
  })

  it('renders log rows with property key chips, filters by level and search query, and expands SQL and JSON details on click', async () => {
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

    const rows = wrapper.findAll('.log-row')
    expect(rows).toHaveLength(3)
    expect(rows[0]?.text()).toContain('Recognition failed')
    expect(rows[0]?.text()).toContain('error')
    expect(rows[2]?.text()).toContain('SQL query')
    expect(rows[2]?.findAll('.log-row__key-chip').map((chip) => chip.text())).toStrictEqual([
      'durationMs',
      'pool',
      'sql',
    ])

    // Expand the SQL row and verify formatted SQL block
    await rows[2]?.find('.log-row__summary').trigger('click')
    expect(rows[2]?.find('.log-row__code').text()).toContain('SELECT id, path\nFROM notes\nWHERE vault_id = ?\nORDER BY path')

    // Expand the RPC row and verify formatted JSON block
    await rows[1]?.find('.log-row__summary').trigger('click')
    expect(rows[1]?.find('.log-row__code').text()).toContain('{\n  "vaultId": "v1",\n  "title": "Welcome"\n}')

    // Filter by Error level
    const errorBtn = wrapper
      .findAll('.logs-tab__level-btn')
      .find((btn) => btn.text() === 'Error')
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
