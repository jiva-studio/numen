/**
 * The services of the window every plugin gets the same of, with no permission: its commands.
 */
import type { PluginContextHost } from './types'

/** The commands the host offers. */
export function createFrameServices(host: PluginContextHost) {
  const commands = host.getCommands?.()
  return commands ? { commands } : {}
}
