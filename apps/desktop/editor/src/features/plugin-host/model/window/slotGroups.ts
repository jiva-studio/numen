import type { Ref } from 'vue'
import {
  formatCommandId,
  getEntryKey,
  sortSlotEntries,
  type DeclaredSlotEntry,
  type DeclaredSlotPoint,
} from '@numen/plugin-loader'
import { getPaletteId, type DeclaredCommand, type DeclaredSlot } from '@/entities/plugin'
import { readSlotEntries } from './pluginSlots'

/** One entry a slot holds, a view or a command, under the key a stored layout names it by. */
interface SlotContributor {
  readonly key: string
  readonly label: string
  readonly pluginId: string
  readonly group: string
}

/** A slot a plugin draws and the views placed in it, in the order it draws them when nothing is stored. */
interface SlotGroup {
  readonly slot: string
  readonly title: string
  readonly contributors: readonly SlotContributor[]
}

/** What the groups are read from. */
interface SlotGroupsSource {
  readonly slots: Readonly<Ref<readonly DeclaredSlot[] | undefined>>
  readonly commands?: Readonly<Ref<readonly DeclaredCommand[] | undefined>>
  readonly contributions: {
    readonly slotPoints: Readonly<Ref<readonly DeclaredSlotPoint[] | undefined>>
  }
}

const formatTitle = (name: string): string => {
  const words = name
    .replace(/([a-z])([A-Z])/g, '$1 $2')
    .replace(/[.-]/g, ' ')
    .toLowerCase()
  return words.charAt(0).toUpperCase() + words.slice(1)
}

/** The slots a plugin draws, other than a menu, that hold entries, each with the views and commands others place in it; a slot without entries is left out. */
export function createSlotGroups(source: SlotGroupsSource) {
  return (pluginId: string): readonly SlotGroup[] => {
    const titles = new Map(
      (source.commands?.value ?? []).map((one) => [getPaletteId(one), one.title]),
    )
    const getLabel = (entry: DeclaredSlotEntry): string => {
      if (entry.title) return entry.title
      if (!entry.command) return formatTitle(entry.view ?? entry.submenu ?? '')
      const full = entry.command.includes(':')
        ? entry.command
        : formatCommandId(entry.pluginId, entry.command)
      return titles.get(full) ?? formatTitle(entry.command.split(':').pop() ?? '')
    }
    const entries = readSlotEntries(source.slots.value) ?? []
    return (source.contributions.slotPoints.value ?? [])
      .filter((point) => point.pluginId === pluginId && !point.menu)
      .map((point) => ({
        slot: point.slot,
        title: point.title,
        contributors: sortSlotEntries(
          entries.filter((one) => one.slot === point.slot),
          (one) => one,
        ).flatMap((entry) => {
          const key = getEntryKey(entry)
          return key
            ? [
                {
                  key,
                  label: getLabel(entry),
                  pluginId: entry.pluginId,
                  group: entry.group ?? '',
                },
              ]
            : []
        }),
      }))
      .filter((group) => group.contributors.length > 0)
  }
}
