/**
 * The plugin host of the window last mounted, so a test registers what a plugin registers.
 *
 * A test file that needs it mocks the plugin support module with `recordHosts`.
 */
import type { PluginHost } from '@numen/plugin-loader'
import type * as PluginSupportModule from '@/app/plugins/pluginSupport'

export const hosts: { current: PluginHost | undefined } = { current: undefined }

/** The plugin support module, recording the host of each window it supports. */
export const recordHosts = (module: typeof PluginSupportModule): typeof PluginSupportModule => ({
  ...module,
  createPluginSupport: (deps) => {
    const support = module.createPluginSupport(deps)
    hosts.current = support.pluginHost
    return support
  },
})
