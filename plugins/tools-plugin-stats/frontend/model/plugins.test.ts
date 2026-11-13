import { describe, expect, it } from 'vitest'
import type { PluginProcess } from '../types'
import { createMockProcessHost } from '@numen/plugin-sdk/testing'
import { createRunningPlugins, sortProcesses } from './plugins'

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
    const model = createRunningPlugins({
      watch: (handler) => {
        expect(model.phase.value).toBe('loading')
        return host.watch(handler)
      },
    })
    model.start()
    expect(model.phase.value).toBe('ready')

    host.setProcesses([create('org.example.maps', 'Maps', { leaseCount: 1 })])

    expect(model.processes.value.map((one) => one.pluginId)).toStrictEqual(['org.example.maps'])
  })

  it('show none where the window gives no list or refuses it', () => {
    const absent = createRunningPlugins(undefined)
    absent.start()
    const refused = createRunningPlugins({
      watch: () => {
        throw new Error('the plugin lacks required permission "plugins:read"')
      },
    })
    refused.start()

    expect([absent.phase.value, refused.phase.value]).toStrictEqual(['unavailable', 'unavailable'])
  })

  it('stop taking lists once stopped', () => {
    const host = createMockProcessHost()
    const model = createRunningPlugins(host)
    model.start()

    model.stop()
    host.setProcesses([create('org.example.maps', 'Maps', { leaseCount: 1 })])

    expect(model.processes.value).toStrictEqual([])
    expect(host.countWatchers()).toBe(0)
  })
})
