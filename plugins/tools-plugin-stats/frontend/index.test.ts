import { describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import type { PluginProcess } from '@numen/plugin-sdk'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
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
}

const setup = () => {
  const context = createMockPluginContext({ pluginId: 'md.numen.tools.plugin.stats' })
  const ticks: number[] = []
  activate(context, {
    clock: { now: () => 0 },
    ticker: {
      start: () => ticks.push(1),
      stop: () => ticks.pop(),
    },
  })
  const wrapper = mount(context.getRegisteredSlotViews()[0]?.component as never)
  return { context, wrapper, ticks }
}

describe('the running plugins frontend', () => {
  it('registers the status view the manifest declares', () => {
    const { context } = setup()

    expect(context.getRegisteredSlotViews().map((one) => one.id)).toStrictEqual([PLUGINS_VIEW])
  })

  it('counts the plugins the core pushes, and counts again on the next push', async () => {
    const { context, wrapper } = setup()

    expect(wrapper.find('.plugin-stats__figure').text()).toBe('0')
    context.work.processes.setProcesses([MAPS, { ...MAPS, pluginId: 'org.example.atlas' }])
    await nextTick()

    expect(wrapper.find('.plugin-stats__figure').text()).toBe('2')
  })

  it('stops listening and moving the time with the plugin', () => {
    const { context, ticks } = setup()
    expect([context.work.processes.countWatchers(), ticks.length]).toStrictEqual([1, 1])

    context.dispose()

    expect([context.work.processes.countWatchers(), ticks.length]).toStrictEqual([0, 0])
  })
})
