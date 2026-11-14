/**
 * A plugin loader for tests, whose status also answers which plugins are active and starting and why
 * one failed.
 */
import { createPluginLoader, isPluginEnabledValue, type PluginLoaderDeps } from '../loader'
import { createPluginStatus } from '../loading'

export function createTestPluginLoader(deps: PluginLoaderDeps) {
  const status =
    deps.status ??
    createPluginStatus({
      isEnabled: (id) => isPluginEnabledValue(deps.host.getSetting?.(id, 'enabled')),
    })
  const loader = createPluginLoader({ ...deps, status })
  const { activePlugins } = status.state
  return {
    ...loader,
    status: {
      ...loader.status,
      isPluginActive: (id: string) => activePlugins.value.has(id),
      getActivePluginIds: () => [...activePlugins.value.keys()],
      getPluginFailure: status.getPluginFailure,
      isPluginStarting: status.isPluginStarting,
    },
  }
}
