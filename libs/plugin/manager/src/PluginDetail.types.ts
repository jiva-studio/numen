import type { ManagedPlugin } from './plugins'

/** What the plugin page draws; the page keeps no state of the plugin and reads it from here. */
export interface PluginDetailProps {
  readonly plugin: ManagedPlugin
  readonly isEnabled: boolean
  readonly isPending?: boolean
  readonly readme: string
  readonly dataFileCount: number
  readonly problem: string
  readonly canInstall: boolean
  readonly hasSettings: boolean
}
