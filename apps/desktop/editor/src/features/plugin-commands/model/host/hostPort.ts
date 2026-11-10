import type { WhenValues } from '@numen/plugin-loader'
import type { Command } from '@/entities/command'
import type { PluginServices } from './pluginServices'

/** What the commands part of the window says of the window: its own commands and the when values. */
export interface WindowFactsSource {
  readonly getAppCommands: () => readonly Command[]
  readonly getValues: () => WhenValues
}

/** What the window offers its plugin host once the window is composed. */
export interface HostPortParts {
  readonly services: PluginServices
  readonly facts: WindowFactsSource
}

/**
 * The one place the plugin host and the window depend on each other. The host is made first, because
 * the parts of the window are made with it; the window opens this port with the services and facts it
 * made. A read before the port opens is a fault of the composition and throws.
 */
export function createHostPort() {
  let parts: HostPortParts | undefined
  const read = (): HostPortParts => {
    if (parts === undefined) throw new Error('The plugin host was asked before the window was composed')
    return parts
  }
  return {
    getServices: (): PluginServices => read().services,
    getAppCommands: (): readonly Command[] => read().facts.getAppCommands(),
    getValues: (): WhenValues => read().facts.getValues(),
    open: (next: HostPortParts): void => {
      if (parts !== undefined) throw new Error('The plugin host port is open already')
      parts = next
    },
  }
}

export type HostPort = ReturnType<typeof createHostPort>
