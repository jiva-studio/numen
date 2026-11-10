import type { Command } from '@/entities/command'
import type { DeclaredSlot } from '@/entities/plugin'
import { getNamedIcon } from '@/shared/ui/icons'
import { createWhenAvailability, getWhenGroup, type WindowWhenContext } from './whenContext'

const PALETTE_NOTE_SLOT = 'palette.note'

const getRowId = (entry: DeclaredSlot): string => {
  const values = Object.entries(entry.values ?? {})
    .sort(([one], [other]) => one.localeCompare(other))
    .map(([name, value]) => `${name}=${value}`)
  return [entry.command, ...values].join('#')
}

/**
 * The rows the palette lists for the note in front from the entries plugins place in the slot
 * `palette.note`: the command of an entry with its values, under the title the entry gives.
 */
export function createSlotPaletteCommands(
  declared: readonly DeclaredSlot[] | undefined,
  context: WindowWhenContext,
): readonly Command[] {
  const isWhenAvailable = createWhenAvailability(context)
  return (declared ?? []).flatMap((entry): Command[] => {
    if (entry.slot !== PALETTE_NOTE_SLOT || !entry.command || !entry.title) return []
    const { when } = entry
    return [
      {
        id: getRowId(entry),
        command: entry.command,
        ...(entry.values ? { values: entry.values } : {}),
        text: entry.title,
        ...(entry.iconName ? { icon: getNamedIcon(entry.iconName) } : {}),
        group: getWhenGroup(context.whenReader.getCondition(when)),
        ...(when ? { when } : {}),
        isAvailable: (target) => target.isReady && isWhenAvailable(when, target),
      },
    ]
  })
}
