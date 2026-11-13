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
  startedBy: { who: 'core', why: 'list tools' },
  holders: [],
}

const countdown = (one: PluginProcess, now = NOW) => {
  const row = getRow(one, now)
  return [row.countdown, row.countdownHint]
}

describe('the row of a running plugin', () => {
  it('counts down to the stop of an idle plugin and keeps its idle time in the tooltip', () => {
    expect(countdown(IDLE)).toStrictEqual(['1m 55s', 'Stops in 1m 55s; idle for 1m 05s'])
    expect(countdown(IDLE, NOW + 60_000)).toStrictEqual(['55s', 'Stops in 55s; idle for 2m 05s'])
  })

  it('says the plugin is stopping once its stop time has passed', () => {
    expect(countdown(IDLE, NOW + 200_000)).toStrictEqual([
      'stopping',
      'Stops at the next check; the core looks for idle plugins every 30 seconds',
    ])
  })

  it('marks a plugin with calls as in use and an idle one as idle', () => {
    const { idleStopTime: _stop, ...running } = IDLE
    const held = getRow({ ...running, leaseCount: 2 }, NOW)
    const idle = getRow(IDLE, NOW)

    expect([held.isInUse, held.status]).toStrictEqual([true, 'In use'])
    expect([idle.isInUse, idle.status]).toStrictEqual([false, 'Idle'])
  })

  it('shows no time for a plugin in use', () => {
    const { idleStopTime: _stop, ...running } = IDLE

    expect(countdown({ ...running, leaseCount: 1 })).toStrictEqual(['', ''])
  })

  it('shows no time for an idle plugin the core gives no stop time', () => {
    const { idleStopTime: _stop, ...idle } = IDLE

    expect(countdown(idle)).toStrictEqual(['', ''])
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
    expect(getRow(IDLE, NOW).healthText).toBe('The tiles are old')
    expect(getRow({ ...IDLE, health: 'ok', healthText: 'All good' }, NOW).healthText).toBe('')
  })
})
