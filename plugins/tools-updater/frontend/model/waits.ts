import type { JsonValue } from '@numen/plugin-sdk'
import { DEFAULT_INTERVAL_HOURS } from './intervalDefault'

/** The waits before a check is repeated after one that reached no answer, the last repeating. */
export const RETRY_DELAYS_MS: readonly number[] = [10_000, 60_000, 15 * 60_000]

/** The wait after a setting that names the feed changed, which lets the backend apply it first. */
export const SETTLE_DELAY_MS = 1_000

const MS_PER_HOUR = 60 * 60 * 1000

/** The wait between two checks the setting names; the default where it holds no usable number. */
export function getIntervalMs(hours: JsonValue | undefined): number {
  const count = typeof hours === 'number' ? hours : Number(hours)
  return (Number.isFinite(count) && count >= 1 ? count : DEFAULT_INTERVAL_HOURS) * MS_PER_HOUR
}
