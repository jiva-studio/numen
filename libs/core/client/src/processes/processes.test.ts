/** The plugins that run, in the windows' terms. */
import { afterEach, describe, expect, it } from 'vitest'
import { HealthSeverity } from '@numen/protocol'
import { pluginService } from '../clients'
import { installCoreAnswers } from '../testing'
import { createProcessClient } from './processes'

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
  it('arrive with their times in milliseconds, their stop for idleness, their health and the time of the list', async () => {
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
      },
    ])
  })
})

describe('the release of a process', () => {
  it('names the plugin whose backend has finished its work', async () => {
    const released: string[] = []
    restore = installCoreAnswers({
      pluginService: {
        releaseProcess: (request: { pluginId: string }) => {
          released.push(request.pluginId)
          return {}
        },
      },
    })

    await processes.releaseProcess('org.example.atlas')

    expect(released).toStrictEqual(['org.example.atlas'])
  })
})
