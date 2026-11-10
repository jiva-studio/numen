/** What tests and stories take from the slices of the window below the app layer, by the deep paths a test may use. */
export { createNameItem, resolveChoice } from '@/features/command-palette/model/search/lookup'
export type { NameMatch } from '@/features/command-palette/model/search/lookup'
export { getSettingAt } from '@/entities/settings/lib/settingAt'
export { default as SettingsTab } from '@/pages/settings/ui/SettingsTab.vue'
export type { Installation } from '@/pages/settings/types'
export { findTabKindIcon } from '@/entities/tab/lib/icons'
export { createWelcomeActions } from '@/pages/welcome/lib/screen'
export { WELCOME_WORDS } from '@/pages/welcome/words'
