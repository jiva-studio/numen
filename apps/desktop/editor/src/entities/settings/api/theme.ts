/** The theme service of the application, as the window's appearance reads it. */
import { createThemeClient } from '@numen/core-client'
import { theme } from '@/shared/api/clients'

export const themes = createThemeClient(theme)
