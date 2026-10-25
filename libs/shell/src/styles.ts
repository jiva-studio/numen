/**
 * The stylesheet each loaded plugin's frontend carries, by plugin id.
 */
const stylesRegistry = new Map<string, string>()

export function registerPluginStyle(pluginId: string, url: string): void {
  stylesRegistry.set(pluginId, url)
}

export function unregisterPluginStyle(pluginId: string): void {
  stylesRegistry.delete(pluginId)
}

export function getPluginStyle(pluginId: string): string | undefined {
  return stylesRegistry.get(pluginId)
}

export function clearPluginStyles(): void {
  stylesRegistry.clear()
}
