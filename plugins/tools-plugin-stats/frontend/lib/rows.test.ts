import { describe, expect, it } from 'vitest'
import type { PluginProcess } from '../types'
import { getRow } from './rows'

const NOW = 1_700_000_600_000

const IDLE: PluginProcess = {
  pluginId: 'org.example.atlas',
  pluginName: 'Atlas',
  startTime: NOW - 10 * 60_000,
  lastUseTime: NOW - 65_000,
  leaseCount: 0,
  isInProcess: false,
  idleStopTime: NOW - 65_000 + 180_000,
  health: 'warning',
  healthText: 'The tiles are old',
}

const line = (one: PluginProcess, now = NOW) => {
  const row = getRow(one, now, 'en-US')
  return [row.state, row.detail]
}

describe('the row of a running plugin', () => {
  it('counts down to the stop of an idle plugin', () => {
    expect(line(IDLE)).toStrictEqual(['Idle 1m 05s', 'stops in 1m 55s'])
    expect(line(IDLE, NOW + 60_000)).toStrictEqual(['Idle 2m 05s', 'stops in 55s'])
  })

  it('says the plugin is stopping once its stop time has passed', () => {
    const row = getRow(IDLE, NOW + 200_000)

    expect(row.detail).toBe('stopping')
    expect(row.detailHint).toBe(
      'Stops at the next check; the core looks for idle plugins every 30 seconds',
    )
  })

  it('counts the calls that keep a plugin in use instead of a stop', () => {
    const { idleStopTime: _stop, ...running } = IDLE
    const held = { ...running, leaseCount: 2 }

    expect(line(held)).toStrictEqual(['In use', '2 calls'])
    expect(line({ ...held, leaseCount: 1 })).toStrictEqual(['In use', '1 call'])
  })

  it('shows only the idle time of an idle plugin the core gives no stop time', () => {
    const { idleStopTime: _stop, ...idle } = IDLE

    expect(line(idle)).toStrictEqual(['Idle 1m 05s', ''])
  })

  it('puts the id, the time it has run and where it runs in the tooltip', () => {
    expect(getRow(IDLE, NOW).title).toBe('org.example.atlas\nRunning for 10m 00s')
    expect(getRow({ ...IDLE, isInProcess: true }, NOW).title).toBe(
      'org.example.atlas\nRunning for 10m 00s\nRuns inside the core',
    )
  })

  it('names the plugin by its id where the core gives no name', () => {
    expect(getRow({ ...IDLE, pluginName: '' }, NOW).name).toBe('org.example.atlas')
  })

  it('writes the health only while the plugin is not healthy', () => {
    const sick = getRow(IDLE, NOW)
    const healthy = getRow({ ...IDLE, health: 'ok', healthText: '' }, NOW)

    expect([sick.healthText, sick.healthLabel]).toStrictEqual([
      'The tiles are old',
      'The tiles are old',
    ])
    expect([healthy.healthText, healthy.healthLabel]).toStrictEqual(['', 'Healthy'])
  })
})
