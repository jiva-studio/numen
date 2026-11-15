/**
 * The views a plugin registered for its settings form and the values and writes they receive.
 */
import { computed } from 'vue'
import type { PluginInfo } from '@/entities/settings'
import { getRegisteredViews, getSettingValues } from '../lib/settingsViews'
import type { InstallationSettings, PluginFieldsHost } from '../types'
import { createPluginSettings } from './createPluginSettings'

export interface SettingsViewsDeps {
  readonly getSettings: () => InstallationSettings
  readonly getHost: () => Pick<PluginFieldsHost, 'getSettingsViews'>
  readonly getPlugin: () => PluginInfo
}

export function useSettingsViews(deps: SettingsViewsDeps) {
  const pluginSettings = createPluginSettings(
    () => deps.getSettings(),
    () => deps.getPlugin().id,
  )
  const views = computed(() =>
    getRegisteredViews(
      deps.getPlugin().settingsViews ?? [],
      deps.getHost().getSettingsViews(deps.getPlugin().id),
    ),
  )
  const values = computed(() =>
    getSettingValues(deps.getPlugin(), pluginSettings.get, pluginSettings.isRequested),
  )

  return { views, values, setSetting: pluginSettings.set }
}
