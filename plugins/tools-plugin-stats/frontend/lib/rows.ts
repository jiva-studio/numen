/**
 * The rows of the popover: one line per running plugin with its name, its health and what it does
 * at a moment, and the rest in the tooltip of the row.
 */
import type { PluginProcess } from '../types'
import { WORDS as words } from '../words'
import { formatCount, formatDuration } from './format'

/**
 * One line of the popover. `healthText` is empty while the plugin is healthy; `healthLabel` is
 * `Healthy` or the text of the worst item. `state` is `In use` or `Idle 57s`; `detail` is `2 calls`,
 * `stops in 2m 02s`, `stopping` or empty. `title` is the tooltip of the line: the id, how long the
 * plugin has run, and where it runs.
 */
export interface Row {
  readonly id: string
  readonly name: string
  readonly health: PluginProcess['health']
  readonly healthText: string
  readonly healthLabel: string
  readonly isInCore: boolean
  readonly state: string
  readonly detail: string
  readonly detailHint: string
  readonly title: string
}

interface Activity {
  readonly state: string
  readonly detail: string
  readonly detailHint: string
}

function getActivity(one: PluginProcess, now: number, locale?: string): Activity {
  if (one.leaseCount > 0) {
    const unit = one.leaseCount === 1 ? words.call : words.calls
    return {
      state: words.inUse,
      detail: `${formatCount(one.leaseCount, locale)} ${unit}`,
      detailHint: words.callsHint,
    }
  }
  const state = `${words.idle} ${formatDuration(now - one.lastUseTime)}`
  if (one.idleStopTime === undefined) return { state, detail: '', detailHint: '' }
  const left = one.idleStopTime - now
  return left > 0
    ? { state, detail: `${words.stopsIn} ${formatDuration(left)}`, detailHint: '' }
    : { state, detail: words.stopsNow, detailHint: words.stopsNowHint }
}

/** The row of one plugin at the moment `now`, in milliseconds since the Unix epoch. */
export function getRow(one: PluginProcess, now: number, locale?: string): Row {
  const isOk = one.health === 'ok'
  const title = [one.pluginId, `${words.runningFor} ${formatDuration(now - one.startTime)}`]
  if (one.isInProcess) title.push(words.inCoreHint)
  return {
    id: one.pluginId,
    name: one.pluginName || one.pluginId,
    health: one.health,
    healthText: isOk ? '' : one.healthText,
    healthLabel: isOk ? words.healthOk : one.healthText,
    isInCore: one.isInProcess,
    ...getActivity(one, now, locale),
    title: title.join('\n'),
  }
}

/** The rows of the plugins in the order given, at the moment `now`. */
export function getRows(processes: readonly PluginProcess[], now: number, locale?: string): Row[] {
  return processes.map((one) => getRow(one, now, locale))
}
