import type { PluginProcess } from '../types'

const getName = (one: PluginProcess) => one.pluginName || one.pluginId

/** The plugins by name, so a row keeps its place while its use and idle times change. */
export function sortProcesses(processes: readonly PluginProcess[]): PluginProcess[] {
  return [...processes].sort(
    (a, b) => getName(a).localeCompare(getName(b)) || a.pluginId.localeCompare(b.pluginId),
  )
}
