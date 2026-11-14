/** The theme service of the application, as the window's appearance reads it. */
import { createThemeClient, type ThemeClient } from '@numen/core-client'
import type { WindowClients } from '@/shared/api/clients'

/** The themes of the application over `theme`. */
export const createThemes = (theme: WindowClients['theme']): ThemeClient => createThemeClient(theme)

export type { ThemeClient }
