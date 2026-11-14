/** The plugins that run, in the windows' terms. */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { create } from '@bufbuild/protobuf'
import { HealthSeverity, WatchPluginProcessesResponseSchema } from '@numen/protocol'
import { pluginService } from '../clients'
import { installCoreAnswers } from '../testing'
import { createProcessClient, type ProcessRpc } from './processes'

const processes = createProcessClient(pluginService)

let restore: (() => void) | undefined

afterEach(() => {
  restore?.()
  restore = undefined
})

const readFirst = async <T>(stream: AsyncIterable<T>): Promise<T | undefined> => {
  for await (const one of stream) return one
  return undefined
}

describe('the plugins that run', () => {
  it('arrive with their times in milliseconds, their stop for idleness, their health, who started and holds them, and the time of the list', async () => {
    restore = installCoreAnswers({
      pluginService: {
        watchPluginProcesses: async function* () {
          yield {
            processes: [
              {
                pluginId: 'org.example.atlas',
                pluginName: 'Atlas',
                startTimeUnixMs: 1_700_000_000_000n,
                lastUseTimeUnixMs: 1_700_000_060_000n,
                leaseCount: 0,
                isInProcess: false,
                idleStopTimeUnixMs: 1_700_000_240_000n,
                health: HealthSeverity.WARNING,
                healthText: 'The tiles are old',
                startedBy: { who: 'Indexing', why: 'embed notes' },
                holders: [{ who: 'Search', why: 'find places' }],
              },
              {
                pluginId: 'org.example.maps',
                pluginName: 'Maps',
                startTimeUnixMs: 1_700_000_000_000n,
                lastUseTimeUnixMs: 1_700_000_000_000n,
                leaseCount: 2,
                isInProcess: true,
                idleStopTimeUnixMs: 0n,
                health: HealthSeverity.OK,
                healthText: '',
                holders: [
                  { who: 'Search', why: 'find places' },
                  { who: 'Orchard', why: 'call tool' },
                ],
              },
            ],
            timeUnixMs: 1_700_000_100_000n,
          }
        },
      },
    })

    const list = await readFirst(processes.watchProcesses(new AbortController().signal))

    expect(list?.time).toBe(1_700_000_100_000)
    expect(list?.processes).toStrictEqual([
      {
        pluginId: 'org.example.atlas',
        pluginName: 'Atlas',
        startTime: 1_700_000_000_000,
        lastUseTime: 1_700_000_060_000,
        leaseCount: 0,
        isInProcess: false,
        idleStopTime: 1_700_000_240_000,
        health: 'warning',
        healthText: 'The tiles are old',
        startedBy: { who: 'Indexing', why: 'embed notes' },
        holders: [{ who: 'Search', why: 'find places' }],
      },
      {
        pluginId: 'org.example.maps',
        pluginName: 'Maps',
        startTime: 1_700_000_000_000,
        lastUseTime: 1_700_000_000_000,
        leaseCount: 2,
        isInProcess: true,
        health: 'ok',
        healthText: '',
        startedBy: { who: '', why: '' },
        holders: [
          { who: 'Search', why: 'find places' },
          { who: 'Orchard', why: 'call tool' },
        ],
      },
    ])
  })

  it('forwards the abort signal to watchPluginProcesses', async () => {
    const watchPluginProcesses = vi.fn(async function* () {
      yield create(WatchPluginProcessesResponseSchema, {})
    })
    const rpc: ProcessRpc = { watchPluginProcesses }
    const signal = new AbortController().signal

    await readFirst(createProcessClient(rpc).watchProcesses(signal))

    expect(watchPluginProcesses).toHaveBeenCalledWith({}, { signal })
  })
})
