import type { SettingsStore } from '@/entities/settings'
import { getSettingsConfig } from '../lib/settingsConfig'

/** The settings of the application and the plugins as the conditions of the commands read them. */
export const getWindowConfig = (rest: Pick<SettingsStore, 'state'>) =>
  getSettingsConfig([{ id: 'app', settings: rest.state.app.value }, ...rest.state.plugins.value])
