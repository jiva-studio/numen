import type { CommandRegistration } from '@numen/plugin-sdk'
import type { PluginPaletteCommand } from './types'

/** The commands the palette offers: every registered command that does not opt out. */
export function extractPaletteCommands(
  commands: ReadonlyMap<string, { reg: CommandRegistration }>,
): PluginPaletteCommand[] {
  const list: PluginPaletteCommand[] = []
  for (const { reg } of commands.values()) {
    if (reg.canShowInPalette === false) continue
    list.push({
      id: `plugin:${reg.id}`,
      title: reg.title,
      ...(reg.shortcut ? { shortcut: reg.shortcut } : {}),
      isOffered: () => true,
    })
  }
  return list
}
