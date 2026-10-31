/** The theme service of the application, in the terms the appearance commands use. */
import { theme } from '@/shared/clients'
import type { ThemeClient } from './appearance'

export const themeClient: ThemeClient = {
  listThemes: async () => {
    const answer = await theme.listThemes({})
    return {
      themes: answer.themes.map((one) => ({
        name: one.name,
        title: one.title,
        origin: one.origin,
        pluginId: one.pluginId,
        isPinned: one.isPinned,
      })),
      applied: answer.applied,
      mode: answer.mode,
    }
  },
  readTheme: async (name) => (await theme.readTheme({ name })).css,
  writeAppearance: async (name, mode) => (await theme.writeAppearance({ name, mode })).error,
  watchThemes: async function* (signal) {
    for await (const update of theme.watchThemes({}, { signal })) yield update.names
  },
}
