/** The theme service of the application, as the window's appearance reads it. */
import { createStreamReader, createThemeClient, type ThemeClient } from '@numen/core-client'
import { createWindowAppearance as createWorkbenchAppearance } from '@numen/workbench'
import type { WindowClients } from '@/shared/api/clients'

/** The themes of the application over `theme`. */
export const createThemes = (theme: WindowClients['theme']): ThemeClient => createThemeClient(theme)

export type { ThemeClient }

export {
  type WindowAppearance,
  type AppearanceWords,
  INTERFACE_SCALE,
  TEXT_SCALE,
} from '@numen/workbench'

/** The window appearance over the themes of the application, following them with the core's follower. */
export const createWindowAppearance = (
  deps: Omit<Parameters<typeof createWorkbenchAppearance>[0], 'createStreamReader'>,
) => createWorkbenchAppearance({ ...deps, createStreamReader })
