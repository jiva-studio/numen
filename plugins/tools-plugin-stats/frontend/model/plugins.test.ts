import { describe, expect, it } from 'vitest'
import { createMockProcessHost } from '@numen/plugin-sdk/testing'
import type { PluginProcess } from '../types'
import { createRunningPlugins, sortProcesses, TICK_MS, type Ticker } from './plugins'

const CORE_TIME = 1_700_000_600_000

const create = (pluginId: string, pluginName: string, extra: Partial<PluginProcess> = {}) =>
  ({
    pluginId,
    pluginName,
    startTime: 1_700_000_000_000,
    lastUseTime: 1_700_000_000_000,
    leaseCount: 0,
    isInProcess: false,
    health: 'ok',
    healthText: '',
    ...extra,
  }) satisfies PluginProcess

const createTime = () => {
  let monotonic = 500
  const runs = new Map<number, () => void>()
  const ticker: Ticker = {
    start: (run, intervalMs) => {
      expect(intervalMs).toBe(TICK_MS)
      runs.set(runs.size + 1, run)
      return runs.size
    },
    stop: (handle) => runs.delete(handle),
  }
  return {
    clock: { now: () => monotonic },
    ticker,
    pass: (ms: number) => {
      monotonic += ms
      for (const run of runs.values()) run()
    },
    countTicks: () => runs.size,
  }
}

describe('the order of the running plugins', () => {
  it('puts the ones in use first by name, then the idle ones by the time they stop', () => {
    const list = [
      create('org.example.late', 'Late', { idleStopTime: 1_700_000_300_000 }),
      create('org.example.zines', 'Zines', { leaseCount: 1 }),
      create('org.example.soon', 'Soon', { idleStopTime: 1_700_000_100_000 }),
      create('org.example.atlas', 'Atlas', { leaseCount: 3 }),
    ]

    expect(sortProcesses(list).map((one) => one.pluginName)).toStrictEqual([
      'Atlas',
      'Zines',
      'Soon',
      'Late',
    ])
  })
})

describe('the running plugins', () => {
  it('wait for the first list, then take each list the core pushes', () => {
    const host = createMockProcessHost()
    const time = createTime()
    const model = createRunningPlugins({
      host: {
        watch: (handler) => {
          expect(model.phase.value).toBe('loading')
          return host.watch(handler)
        },
      },
      ...time,
    })
    model.start()
    expect(model.phase.value).toBe('ready')

    host.setProcesses([create('org.example.maps', 'Maps', { leaseCount: 1 })], CORE_TIME)

    expect(model.processes.value.map((one) => one.pluginId)).toStrictEqual(['org.example.maps'])
  })

  it('move the time of the core on between two lists, and take the time of the next list', () => {
    const host = createMockProcessHost([], CORE_TIME)
    const time = createTime()
    const model = createRunningPlugins({ host, ...time })
    model.start()
    expect(model.now.value).toBe(CORE_TIME)

    time.pass(3_000)
    expect(model.now.value).toBe(CORE_TIME + 3_000)

    host.setProcesses([], CORE_TIME + 3_500)
    expect(model.now.value).toBe(CORE_TIME + 3_500)
  })

  it('keep the time of a list sent again unchanged', () => {
    const host = createMockProcessHost([], CORE_TIME)
    const time = createTime()
    const model = createRunningPlugins({ host, ...time })
    model.start()
    time.pass(2_000)

    host.setProcesses([], CORE_TIME)

    expect(model.now.value).toBe(CORE_TIME + 2_000)
  })

  it('show none where the window gives no list or refuses it', () => {
    const absent = createRunningPlugins({ host: undefined, ...createTime() })
    absent.start()
    const refused = createRunningPlugins({
      host: {
        watch: () => {
          throw new Error('the plugin lacks required permission "plugins:read"')
        },
      },
      ...createTime(),
    })
    refused.start()

    expect([absent.phase.value, refused.phase.value]).toStrictEqual(['unavailable', 'unavailable'])
  })

  it('stop taking lists and moving the time once stopped', () => {
    const host = createMockProcessHost()
    const time = createTime()
    const model = createRunningPlugins({ host, ...time })
    model.start()

    model.stop()
    host.setProcesses([create('org.example.maps', 'Maps', { leaseCount: 1 })])

    expect(model.processes.value).toStrictEqual([])
    expect(host.countWatchers()).toBe(0)
    expect(time.countTicks()).toBe(0)
  })
})
