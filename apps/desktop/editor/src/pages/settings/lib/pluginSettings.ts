import { isShown, type Consent, type PluginInfo } from '@/entities/settings'

/** Whether the plugin declares anything its page draws: settings, views of its own, or a secret that is shown. */
export function hasPluginSettings(plugin: PluginInfo, read: (key: string) => unknown): boolean {
  return (
    plugin.settings.length > 0 ||
    (plugin.settingsViews ?? []).length > 0 ||
    plugin.secrets.some((one) => isShown(plugin, one.when, read))
  )
}

/**
 * The consent a plugin's page shows: the one the host reports, or, where the settings hold
 * `aiConsent`, the one that setting names, a value requested and not yet read back among them.
 */
export function getShownConsent(plugin: PluginInfo, read: (key: string) => unknown): Consent {
  const isAllowed = read('aiConsent')
  if (plugin.consent === 'none' || typeof isAllowed !== 'boolean') return plugin.consent
  return isAllowed ? 'allowed' : 'denied'
}
