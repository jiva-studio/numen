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

const facts = (one: PluginProcess, now = NOW) =>
  getRow(one, now, 'en-US').facts.map((fact) => [fact.label, fact.value])

describe('the row of a running plugin', () => {
  it('counts down to the stop of an idle plugin', () => {
    expect(facts(IDLE)).toStrictEqual([
      ['Running for', '10m 00s'],
      ['Idle for', '1m 05s'],
      ['Stops in', '1m 55s'],
    ])
    expect(facts(IDLE, NOW + 60_000)).toContainEqual(['Stops in', '55s'])
  })

  it('says the stop is at the next check once its time has passed', () => {
    expect(facts(IDLE, NOW + 200_000)).toContainEqual(['Stops in', 'Stops at the next check'])
  })

  it('names what keeps a plugin in use instead of a stop', () => {
    const { idleStopTime: _stop, ...running } = IDLE
    const held = { ...running, leaseCount: 2 }

    expect(facts(held)).toStrictEqual([
      ['Running for', '10m 00s'],
      ['In use', '2 calls, streams or tasks'],
    ])
    expect(facts({ ...held, leaseCount: 1 })).toContainEqual(['In use', '1 call, stream or task'])
  })

  it('names the plugin by its id where the core gives no name, and shows its health', () => {
    const row = getRow({ ...IDLE, pluginName: '' }, NOW)

    expect(row.name).toBe('org.example.atlas')
    expect(row.healthText).toBe('The tiles are old')
    expect(getRow({ ...IDLE, health: 'ok', healthText: '' }, NOW).healthText).toBe('Healthy')
  })
})
