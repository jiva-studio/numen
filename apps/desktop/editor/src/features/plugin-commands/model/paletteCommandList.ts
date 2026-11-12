/**
 * The commands the palette lists besides the ones it owns: those of the features, the settings,
 * the declarations of the plugins, the entries of the slots and the commands the plugins registered.
 */
import type { Command } from '@/entities/command'
import { createSlotPaletteCommands } from '../lib/slotPaletteCommands'
import type { DeclaredCommands } from './declaredCommands'

export interface PaletteCommandSources {
  readonly getAppCommands: () => readonly Command[]
  readonly getSettingCommands: () => readonly Command[]
  readonly declaredCommands: Pick<DeclaredCommands, 'paletteCommands' | 'slots' | 'whenContext'>
  readonly getPluginEntries: () => readonly Command[]
}

export function createPaletteCommandList(sources: PaletteCommandSources) {
  const { declaredCommands } = sources
  /** The commands of the features and the ones the plugins declare for the palette. */
  const listAppCommands = (): readonly Command[] => [
    ...sources.getAppCommands(),
    ...declaredCommands.paletteCommands.value,
  ]
  return {
    listAppCommands,
    listCommands: (): readonly Command[] => [
      ...sources.getAppCommands(),
      ...sources.getSettingCommands(),
      ...declaredCommands.paletteCommands.value,
      ...createSlotPaletteCommands(declaredCommands.slots.value, declaredCommands.whenContext),
      ...sources.getPluginEntries(),
    ],
  }
}
