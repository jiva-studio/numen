import { formatCommandId } from '@numen/plugin-loader'

/** What the palette files a declared command under. */
export const getPaletteId = (command: { readonly pluginId: string; readonly id: string }): string =>
  formatCommandId(command.pluginId, command.id)
