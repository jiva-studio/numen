import type { PluginLifecycle } from './types'

// definePlugin preserves strict typing on a plugin definition.
export function definePlugin(plugin: PluginLifecycle): PluginLifecycle {
  return plugin
}
