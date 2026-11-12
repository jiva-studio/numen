import type { SettingsStore } from '@/entities/settings'

/** What the settings list of each plugin: its name, and whether it has a frontend or a backend. */
export function createPluginCatalog(rest: Pick<SettingsStore, 'state'>) {
  const find = (pluginId: string) => rest.state.plugins.value.find((one) => one.id === pluginId)
  return {
    getName: (pluginId: string): string => find(pluginId)?.name ?? pluginId,
    hasFrontend: (pluginId: string): boolean => find(pluginId)?.hasFrontend ?? true,
    hasBackend: (pluginId: string): boolean => find(pluginId)?.hasBackend ?? true,
    getReaderExtensions: (pluginId: string, readerId: string): readonly string[] =>
      find(pluginId)?.readers?.find((reader) => reader.id === readerId)?.extensions ?? [],
  }
}
