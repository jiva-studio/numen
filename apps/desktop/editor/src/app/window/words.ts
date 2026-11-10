/**
 * What the window reports about plugins and the vault list.
 */
// Stryker disable StringLiteral: interface copy

const toLowerFirst = (text: string): string => text.charAt(0).toLowerCase() + text.slice(1)

export const WINDOW_WORDS = {
  /** The name the title bar of every window starts with. */
  appName: 'numen',
  /** The list of vaults did not answer. */
  unlistedVaults: 'Vault list unavailable',
  /** Plugin failures, shown as failed tasks: "<Name> stopped: <reason>", and errors raised by plugin code. */
  pluginStopped: (name: string, reason: string) => `${name} stopped: ${toLowerFirst(reason)}`,
  pluginRetryFailed: (name: string, message: string) => `Could not retry ${name}: ${message}`,
  pluginError: (name: string, message: string) => `${name}: ${message}`,
  pluginLoadFailed: 'failed to start',
}
