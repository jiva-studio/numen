/** How a plugin stands: switched off, offered by a source and not installed, or installed and enabled. */
export type PluginStanding = 'disabled' | 'offered' | 'installed'

/**
 * The plugin that provides a kind of tab, as the placeholder of the kind needs it. The feature is
 * the one the plugin declares, else its name. The incompatibility says why the application cannot
 * run the plugin and is null where it can. The kind title is the one the plugin declares for the
 * kind, empty where it declares none.
 */
export interface TabKindPlugin {
  readonly id: string
  readonly name: string
  readonly feature: string
  readonly standing: PluginStanding
  readonly incompatibility: string | null
  readonly kindTitle: string
}

/** What the catalog answered for the plugin that provides a kind of tab. */
export type UnavailableLookup =
  | { readonly status: 'pending' }
  | { readonly status: 'found'; readonly plugin: TabKindPlugin }
  | { readonly status: 'missing' }

/** What the person can do about the plugin: switch it on, install it, or look at it. */
type UnavailableAction = 'enable' | 'install' | 'inspect'

/** The words and the action of the placeholder of one tab. */
export interface UnavailableView {
  readonly heading: string
  readonly message: string
  readonly action: UnavailableAction
  readonly actionText: string
  readonly pluginId: string | undefined
}

const MISSING: UnavailableView = {
  heading: 'Plugin unavailable',
  message: 'The plugin for this tab is disabled or missing.',
  action: 'inspect',
  actionText: 'Open Settings',
  pluginId: undefined,
}

const createIncompatibleView = (plugin: TabKindPlugin, detail: string): UnavailableView => ({
  heading: 'Plugin incompatible',
  message: `${plugin.name} cannot run in this application: ${detail}`,
  action: 'inspect',
  actionText: 'Open Settings',
  pluginId: plugin.id,
})

const createDisabledView = (plugin: TabKindPlugin, isEnabling: boolean): UnavailableView => ({
  heading: 'Plugin disabled',
  message: `The plugin for ${plugin.feature} is disabled.`,
  action: 'enable',
  actionText: isEnabling ? `Enabling ${plugin.name}…` : `Enable ${plugin.name}`,
  pluginId: plugin.id,
})

const createOfferedView = (plugin: TabKindPlugin): UnavailableView => ({
  heading: 'Plugin not installed',
  message: `The plugin for ${plugin.feature} is not installed.`,
  action: 'install',
  actionText: `Install ${plugin.name}`,
  pluginId: plugin.id,
})

const createNotRunning = (plugin: TabKindPlugin): UnavailableView => ({
  heading: 'Plugin not running',
  message: `The plugin for ${plugin.feature} is not running.`,
  action: 'inspect',
  actionText: 'Open Settings',
  pluginId: plugin.id,
})

/**
 * What the placeholder of a tab says, read from the plugin that provides its kind. A lookup the
 * catalog has not answered has no view.
 */
export function getUnavailableView(
  lookup: UnavailableLookup,
  isEnabling: boolean,
): UnavailableView | undefined {
  if (lookup.status === 'pending') return undefined
  if (lookup.status === 'missing') return MISSING
  const { plugin } = lookup
  if (plugin.incompatibility !== null) return createIncompatibleView(plugin, plugin.incompatibility)
  if (plugin.standing === 'disabled') return createDisabledView(plugin, isEnabling)
  if (plugin.standing === 'offered') return createOfferedView(plugin)
  return createNotRunning(plugin)
}
