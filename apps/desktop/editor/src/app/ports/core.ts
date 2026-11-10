import type { SettingsPort } from './settings'
import type { VaultPort } from './vault'

export interface Core extends SettingsPort, VaultPort {}
