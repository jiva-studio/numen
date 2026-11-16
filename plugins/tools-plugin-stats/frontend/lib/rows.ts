import type { PluginProcess } from '../types'
import { getRow, type Row } from './row'

/** The rows of the plugins in the order given, at the moment `now`. */
export function getRows(processes: readonly PluginProcess[], now: number): Row[] {
  return processes.map((one) => getRow(one, now))
}
