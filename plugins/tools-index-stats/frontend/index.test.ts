import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import type { Stats } from './api/stats'
import { activate, STATS_VIEW } from './index'
import type { Timer } from './model/stats'

const STATS: Stats = {
  vectors: 12480,
  chunks: 13100,
  waiting: 620,
  sources: 84,
  kinds: [],
  model: 'bge-m3',
  dimensions: 1024,
  profile: 'balanced',
  threads: 4,
  databaseBytes: 0,
  vectorBytes: 0,
}

const flush = async () => {
  for (let i = 0; i < 5; i++) await Promise.resolve()
}

const setup = (answer: Stats) => {
  const waits: Array<{ run: () => void; delayMs: number; id: number }> = []
  const timer: Timer = {
    set: (run, delayMs) => {
      waits.push({ run, delayMs, id: waits.length })
      return waits.length - 1
    },
    clear: (handle) => {
      const index = waits.findIndex((one) => one.id === handle)
      if (index !== -1) waits.splice(index, 1)
    },
  }
  const get = vi.fn(async () => ({ ok: true as const, value: answer }))
  const context = createMockPluginContext({ pluginId: 'md.numen.tools.index.stats' })
  activate(context, {
    api: { get },
    timer,
    activity: { isActive: () => true, onActive: () => () => undefined },
  })
  return { context, get, waits }
}

describe('the index stats frontend', () => {
  it('registers the status view the manifest declares', () => {
    const { context } = setup(STATS)

    expect(STATS_VIEW).toBe('stats')
    expect(context.getRegisteredSlotViews().map((one) => one.id)).toEqual(['stats'])
  })

  it('asks the backend when it starts and draws the count', async () => {
    const { context, get } = setup(STATS)
    await flush()

    const wrapper = mount(context.getRegisteredSlotViews()[0]?.component as never)

    expect(get).toHaveBeenCalledTimes(1)
    expect(wrapper.find('button').text()).toMatch(/^⬡12.480$/)
  })

  it('draws the new count when the next ask returns one', async () => {
    const { context, waits } = setup(STATS)
    await flush()
    const wrapper = mount(context.getRegisteredSlotViews()[0]?.component as never)

    expect(waits.map((one) => one.delayMs)).toEqual([2000])
    expect(wrapper.find('button').exists()).toBe(true)
  })

  it('stops asking with the plugin', async () => {
    const { context, waits } = setup(STATS)
    await flush()

    context.dispose()

    expect(waits).toEqual([])
  })
})
