// Stryker disable StringLiteral: interface copy

const toLowerFirst = (text: string): string => text.charAt(0).toLowerCase() + text.slice(1)

/** What the window reports about a plugin that failed, shown as failed tasks and as errors of plugin code. */
export const PLUGIN_FAILURE_WORDS = {
  /** "<Name> stopped: <reason>". */
  stopped: (name: string, reason: string) => `${name} stopped: ${toLowerFirst(reason)}`,
  retryFailed: (name: string, message: string) => `Could not retry ${name}: ${message}`,
  error: (name: string, message: string) => `${name}: ${message}`,
  loadFailed: 'failed to start',
}
