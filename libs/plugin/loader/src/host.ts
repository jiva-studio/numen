import type { Disposable, JsonValue, PluginContext } from '@numen/plugin-sdk'
import type { Manifest } from '@numen/core-client'

/** What the loader requires of the window that hosts the plugins it activates. */
export interface LoaderHost {
  createContext(pluginId: string, manifest?: Manifest): PluginContext
  /** Called before the context is made: the plugin counts as started only after `markPluginStarted`. */
  markPluginStarting?(pluginId: string): void
  /** Called once `activate` has returned. */
  markPluginStarted?(pluginId: string): void
  disposePlugin(pluginId: string): void
  disposeAll(): void
  registerManifest?(pluginId: string, manifest: Manifest): void
  getSetting?(pluginId: string, key: string): JsonValue | undefined
  onSettingChange?<T extends JsonValue = JsonValue>(
    pluginId: string,
    key: string,
    handler: (value: T) => void | Promise<void>,
  ): Disposable
}
