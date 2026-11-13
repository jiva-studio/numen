/**
 * The rows of the popover: one per running plugin, with its name, its health and the facts of its
 * process at a moment.
 */
import type { PluginProcess } from '../types'
import { WORDS as words } from '../words'
import { formatCount, formatDuration } from './format'

/** One fact of a plugin: a label, a value and, where the meaning is not obvious, a one-line hint. */
export interface Fact {
  readonly id: string
  readonly label: string
  readonly value: string
  readonly hint: string
}

export interface Row {
  readonly id: string
  readonly name: string
  readonly health: PluginProcess['health']
  readonly healthText: string
  readonly isInCore: boolean
  readonly facts: readonly Fact[]
}

const fact = (id: string, label: string, value: string, hint = ''): Fact => ({
  id,
  label,
  value,
  hint,
})

function getUseFacts(one: PluginProcess, now: number, locale?: string): Fact[] {
  if (one.leaseCount > 0 || one.idleStopTime === undefined) {
    const unit = one.leaseCount === 1 ? words.lease : words.leases
    return [fact('in-use', words.inUse, `${formatCount(one.leaseCount, locale)} ${unit}`)]
  }
  const left = one.idleStopTime - now
  return [
    fact('idle-for', words.idleFor, formatDuration(now - one.lastUseTime)),
    left > 0
      ? fact('stops-in', words.stopsIn, formatDuration(left))
      : fact('stops-in', words.stopsIn, words.stopsNow, words.stopsNowHint),
  ]
}

/** The row of one plugin at the moment `now`, in milliseconds since the Unix epoch. */
export function getRow(one: PluginProcess, now: number, locale?: string): Row {
  return {
    id: one.pluginId,
    name: one.pluginName || one.pluginId,
    health: one.health,
    healthText: one.health === 'ok' ? words.healthOk : one.healthText,
    isInCore: one.isInProcess,
    facts: [
      fact('running-for', words.runningFor, formatDuration(now - one.startTime)),
      ...getUseFacts(one, now, locale),
    ],
  }
}

/** The rows of the plugins in the order given, at the moment `now`. */
export function getRows(processes: readonly PluginProcess[], now: number, locale?: string): Row[] {
  return processes.map((one) => getRow(one, now, locale))
}
