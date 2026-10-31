import type { Disposable, PluginContext, Manifest } from '@numen/plugin-sdk'

/** What the loader requires of the window that hosts the plugins it activates. */
export interface LoaderHost {
  createContext(pluginId: string, manifest?: Manifest): PluginContext
  disposePlugin(pluginId: string): void
  disposeAll(): void
  registerManifest?(pluginId: string, manifest: Manifest): void
  getSetting?(pluginId: string, key: string): unknown
  onSettingChange?<T = unknown>(
    pluginId: string,
    key: string,
    handler: (value: T) => void | Promise<void>,
  ): Disposable
}
