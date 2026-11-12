import type { SettingsStore } from '@/entities/settings'
import { getSettingsConfig } from '../lib/settingsConfig'

/** The settings of the application and the plugins as the conditions of the commands read them. */
export const getWindowConfig = (rest: Pick<SettingsStore, 'state'>) =>
  getSettingsConfig([
    { id: 'app', name: '', settings: rest.state.app.value },
    ...rest.state.plugins.value.map((one) => ({
      id: one.id,
      name: one.name,
      settings: one.settings,
    })),
  ])
