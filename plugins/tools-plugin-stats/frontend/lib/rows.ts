/**
 * The rows of the popover: one line per running plugin with whether it is in use, its name, the
 * worst health item of an unhealthy plugin and the time until an idle plugin stops, and the rest in
 * the tooltips.
 */
import type { PluginProcess } from '../types'
import { WORDS as words } from '../words'
import { formatDuration } from './format'

/**
 * One line of the popover. `isInUse` is true while the plugin holds calls, streams or tasks;
 * `status` is `In use` or `Idle`. `healthText` is empty while the plugin is healthy. `countdown` is
 * `2m 02s` until an idle plugin stops, `stopping` once that time has passed, and empty for a plugin
 * in use; `countdownHint` is its tooltip. `title` is the tooltip of the line: the id, how long the
 * plugin has run, and where it runs.
 */
export interface Row {
  readonly id: string
  readonly name: string
  readonly isInUse: boolean
  readonly status: string
  readonly health: PluginProcess['health']
  readonly healthText: string
  readonly isInCore: boolean
  readonly countdown: string
  readonly countdownHint: string
  readonly title: string
}

interface Countdown {
  readonly countdown: string
  readonly countdownHint: string
}

const NO_COUNTDOWN: Countdown = { countdown: '', countdownHint: '' }

function getCountdown(one: PluginProcess, now: number): Countdown {
  if (one.leaseCount > 0 || one.idleStopTime === undefined) return NO_COUNTDOWN
  const left = one.idleStopTime - now
  if (left <= 0) return { countdown: words.stopsNow, countdownHint: words.stopsNowHint }
  const countdown = formatDuration(left)
  const idle = formatDuration(now - one.lastUseTime)
  return { countdown, countdownHint: `${words.stopsIn} ${countdown}; ${words.idleFor} ${idle}` }
}

/** The row of one plugin at the moment `now`, in milliseconds since the Unix epoch. */
export function getRow(one: PluginProcess, now: number): Row {
  const isInUse = one.leaseCount > 0
  const title = [one.pluginId, `${words.runningFor} ${formatDuration(now - one.startTime)}`]
  if (one.isInProcess) title.push(words.inCoreHint)
  return {
    id: one.pluginId,
    name: one.pluginName || one.pluginId,
    isInUse,
    status: isInUse ? words.inUse : words.idle,
    health: one.health,
    healthText: one.health === 'ok' ? '' : one.healthText,
    isInCore: one.isInProcess,
    ...getCountdown(one, now),
    title: title.join('\n'),
  }
}

/** The rows of the plugins in the order given, at the moment `now`. */
export function getRows(processes: readonly PluginProcess[], now: number): Row[] {
  return processes.map((one) => getRow(one, now))
}
