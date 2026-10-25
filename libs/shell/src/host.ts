import type { Disposable, PluginContext, PluginManifest } from '@numen/plugin-sdk'

/** What the loader asks of the window that hosts the plugins it activates. */
export interface LoaderHost {
  createContext(pluginId: string, manifest?: PluginManifest): PluginContext
  disposePlugin(pluginId: string): void
  disposeAll(): void
  registerManifest?(pluginId: string, manifest: PluginManifest): void
  getSetting?(pluginId: string, key: string): unknown
  onSettingChange?<T = unknown>(
    pluginId: string,
    key: string,
    handler: (value: T) => void | Promise<void>,
  ): Disposable
}
