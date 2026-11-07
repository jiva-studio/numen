/**
 * The services of the window every plugin gets the same of: its commands, its vaults and the window
 * itself, with no permission.
 */
import type { PluginContextHost } from './types'

/** The commands, the vaults and the window the host offers. */
export function createFrameServices(host: PluginContextHost) {
  const commands = host.getCommands?.()
  const vaults = host.getVaults?.()
  const window = host.getWindow?.()
  return {
    ...(commands ? { commands } : {}),
    ...(vaults ? { vaults } : {}),
    ...(window ? { window } : {}),
  }
}
