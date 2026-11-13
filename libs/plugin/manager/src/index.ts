/** The plugin list and the plugin page: data in by props, answers out as events. */
import './styles.css'

export { default as PluginsList } from './PluginsList.vue'
export { default as PluginDetail } from './PluginDetail.vue'
export { default as NavigationRow } from './NavigationRow.vue'
export { default as SettingsToolbar } from './SettingsToolbar.vue'
export { default as PluginSettingsCard } from './PluginSettingsCard.vue'
export type { PluginDetailProps } from './PluginDetail.types'
export {
  createManagedPlugins,
  isInstalled,
  OTHER_CATEGORY,
  type ManagedPlugin,
  type PluginListing,
} from './plugins'
export type {
  PluginContribution,
  PluginContributionItem,
  PluginContributionKind,
  PluginHealth,
  PluginInfo,
  PluginReader,
  PluginSettingCondition,
  PluginSettingInfo,
  SettingValue,
  PluginSettingOption,
  PluginSettingOptionsBy,
  PluginSettingsViewInfo,
  PluginSettingType,
} from './pluginInfo'
export {
  formatCategoryLabel,
  getFilteredPluginItems,
  getPluginCategory,
  matchesPluginQuery,
  parsePluginQuery,
  type FlatPluginItem,
  type ParsedPluginQuery,
  type PluginFilterContext,
} from './filter'
export {
  getPluginState,
  getStartItem,
  getWorstItem,
  START_HEALTH_KEY,
  type PluginState,
} from './state'
export { getReadmeText } from './readme'
export { PLUGIN_WORDS } from './words'
