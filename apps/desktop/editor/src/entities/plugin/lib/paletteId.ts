import { formatCommandId } from '@numen/plugin-loader'
import type { DeclaredCommand } from '../api/declaredCommands'

/** What the palette files a declared command under. */
export const getPaletteId = (command: DeclaredCommand): string =>
  formatCommandId(command.pluginId, command.id)
