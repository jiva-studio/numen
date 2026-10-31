import type { ManagedPlugin } from './plugins'

/** What the plugin page draws; the page keeps no state of the plugin and reads it from here. */
export interface PluginDetailProps {
  readonly plugin: ManagedPlugin
  /** Whether the plugin is switched on. */
  readonly isEnabled: boolean
  /** The README.md of the plugin; empty where it has none. */
  readonly readme: string
  /** How many files the plugin keeps in the shown vault; zero hides the delete action. */
  readonly dataFileCount: number
  /** Why the last attempt to delete the data failed; empty otherwise. */
  readonly problem: string
  /** Whether the application can install a plugin that is not installed. */
  readonly canInstall: boolean
  /** Whether the `settings` slot holds anything to draw. */
  readonly hasSettings: boolean
}
