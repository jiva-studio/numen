/**
 * A plugin loader for tests, whose status also answers which plugins are active and starting and why
 * one failed.
 */
import { createPluginLoader, isPluginEnabledValue, type PluginLoaderDeps } from '../loader'
import { createPluginStatus } from '../loading'
import { createPluginStyles, type PluginStyles } from '../styles'

/** The deps of a loader, with a stylesheet store of its own where the test gives none. */
type TestLoaderDeps = Omit<PluginLoaderDeps, 'styles'> & { readonly styles?: PluginStyles }

export function createTestPluginLoader(deps: TestLoaderDeps) {
  const status =
    deps.status ??
    createPluginStatus({
      isEnabled: (id) => isPluginEnabledValue(deps.host.getSetting?.(id, 'enabled')),
    })
  const loader = createPluginLoader({ styles: createPluginStyles(), ...deps, status })
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
