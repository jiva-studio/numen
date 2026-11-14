import { describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import type { PluginProcess } from '@numen/plugin-sdk'
import { createMockHostServices, createMockPluginContext } from '@numen/plugin-sdk/testing'
import { activate, PLUGINS_VIEW } from './index'

const MAPS: PluginProcess = {
  pluginId: 'org.example.maps',
  pluginName: 'Maps',
  startTime: 1_700_000_000_000,
  lastUseTime: 1_700_000_000_000,
  leaseCount: 1,
  isInProcess: false,
  health: 'ok',
  healthText: '',
  startedBy: { who: 'Settings', why: 'list tools' },
  holders: [],
}

const setup = () => {
  const context = createMockPluginContext({ pluginId: 'md.numen.tools.plugin.stats' })
  const host = createMockHostServices()
  const ticks: number[] = []
  activate(context, host, {
    clock: { now: () => 0 },
    ticker: {
      start: () => ticks.push(1),
      stop: () => ticks.pop(),
    },
  })
  const wrapper = mount(context.getRegisteredSlotViews()[0]?.component as never)
  return { context, host, wrapper, ticks }
}

describe('the running plugins frontend', () => {
  it('registers the status view the manifest declares', () => {
    const { context } = setup()

    expect(context.getRegisteredSlotViews().map((one) => one.id)).toStrictEqual([PLUGINS_VIEW])
  })

  it('counts the plugins the core pushes, and counts again on the next push', async () => {
    const { host, wrapper } = setup()

    expect(wrapper.find('.plugin-stats__figure').text()).toBe('0')
    host.work.processes.setProcesses([MAPS, { ...MAPS, pluginId: 'org.example.atlas' }])
    await nextTick()

    expect(wrapper.find('.plugin-stats__figure').text()).toBe('2')
  })

  it('stops listening and moving the time with the plugin', () => {
    const { context, host, ticks } = setup()
    expect([host.work.processes.countWatchers(), ticks.length]).toStrictEqual([1, 1])

    context.dispose()

    expect([host.work.processes.countWatchers(), ticks.length]).toStrictEqual([0, 0])
  })
})
