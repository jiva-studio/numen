/**
 * The commands the palette lists besides the ones it owns: those of the features, the settings,
 * the declarations of the plugins, the entries of the slots and the commands the plugins registered.
 *
 * The commands of the plugins are listed in the order their manifests declare them, whichever of the
 * backend or the frontend of a plugin offers them, so the list holds its order while the plugins start.
 */
import type { Command } from '@/entities/command'
import { getPaletteId } from '@/entities/plugin'
import { createSlotPaletteCommands } from '../lib/slotPaletteCommands'
import type { DeclaredCommands } from './declaredCommands'

export interface PaletteCommandSources {
  readonly getAppCommands: () => readonly Command[]
  readonly getSettingCommands: () => readonly Command[]
  readonly declaredCommands: Pick<
    DeclaredCommands,
    'paletteCommands' | 'slots' | 'whenContext' | 'commands'
  >
  readonly getPluginEntries: () => readonly Command[]
}

export function createPaletteCommandList(sources: PaletteCommandSources) {
  const { declaredCommands } = sources
  /** The commands of the features and the ones the plugins declare for the palette. */
  const listAppCommands = (): readonly Command[] => [
    ...sources.getAppCommands(),
    ...declaredCommands.paletteCommands.value,
  ]
  const listPluginCommands = (): readonly Command[] => {
    const order = new Map(
      (declaredCommands.commands.value ?? []).map((one, at) => [getPaletteId(one), at]),
    )
    const getPosition = (command: Command): number => order.get(command.id) ?? order.size
    return [...declaredCommands.paletteCommands.value, ...sources.getPluginEntries()].sort(
      (one, other) => getPosition(one) - getPosition(other) || one.id.localeCompare(other.id),
    )
  }
  return {
    listAppCommands,
    listCommands: (): readonly Command[] => [
      ...sources.getAppCommands(),
      ...sources.getSettingCommands(),
      ...listPluginCommands(),
      ...createSlotPaletteCommands(declaredCommands.slots.value, declaredCommands.whenContext),
    ],
  }
}
