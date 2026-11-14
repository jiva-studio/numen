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
  recipe: 'local|bge-m3|1024|512|mean|int8@0.4',
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

  it('asks the backend once when it starts, draws the count and sets no timer', async () => {
    const { context, get, waits } = setup(STATS)
    await flush()

    const wrapper = mount(context.getRegisteredSlotViews()[0]?.component as never)

    expect(get).toHaveBeenCalledTimes(1)
    expect(waits).toEqual([])
    expect(wrapper.find('.index-stats__trigger').text()).toMatch(/^⬡12.480$/)
  })

  it('asks the backend when a note of the vault changes', async () => {
    const { context, get } = setup(STATS)
    await flush()

    context.events.emitVaultChange({ paths: ['Garden/Roses.md'] })
    await flush()

    expect(get).toHaveBeenCalledTimes(2)
  })

  it('asks the backend when a background task of the window starts', async () => {
    const { context, get } = setup(STATS)
    await flush()

    context.work.backgroundTasks.setTasks([
      {
        id: 'indexing',
        label: 'Indexing',
        about: '',
        state: 'running',
        progress: { completedCount: 0, total: 0, counting: '' },
        error: '',
        allowed: { cancel: true, retry: false },
        actionLabel: '',
      },
    ])
    await flush()

    expect(get).toHaveBeenCalledTimes(2)
  })

  it('asks every two seconds while the popover is open, and stops when it closes', async () => {
    const { context, waits } = setup(STATS)
    await flush()
    const wrapper = mount(context.getRegisteredSlotViews()[0]?.component as never, {
      attachTo: document.body,
    })

    await wrapper.find('.index-stats__trigger').trigger('click')
    expect(waits.map((one) => one.delayMs)).toEqual([2000])

    await wrapper.find('.index-stats__trigger').trigger('click')
    expect(waits).toEqual([])
    wrapper.unmount()
  })

  it('stops asking when the view goes with the popover open', async () => {
    const { context, waits } = setup(STATS)
    await flush()
    const wrapper = mount(context.getRegisteredSlotViews()[0]?.component as never, {
      attachTo: document.body,
    })
    await wrapper.find('.index-stats__trigger').trigger('click')

    wrapper.unmount()

    expect(waits).toEqual([])
  })

  it('stops asking with the plugin', async () => {
    const { context, waits } = setup(STATS)
    await flush()

    context.dispose()

    expect(waits).toEqual([])
  })
})
