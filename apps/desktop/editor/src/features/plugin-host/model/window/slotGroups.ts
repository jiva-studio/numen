import type { Ref } from 'vue'
import { getEntryKey, sortSlotEntries, type DeclaredSlotPoint } from '@numen/plugin-loader'
import type { DeclaredSlot } from '@/entities/plugin'
import { readSlotEntries } from './pluginSlots'

/** One view a slot holds, under the key a stored layout names it by. */
interface SlotContributor {
  readonly key: string
  readonly label: string
  readonly pluginId: string
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
  readonly contributions: {
    readonly slotPoints: Readonly<Ref<readonly DeclaredSlotPoint[] | undefined>>
  }
}

const prettify = (name: string): string => {
  const words = name
    .replace(/([a-z])([A-Z])/g, '$1 $2')
    .replace(/[.-]/g, ' ')
    .toLowerCase()
  return words.charAt(0).toUpperCase() + words.slice(1)
}

/** The slots a plugin draws that hold views, each with the views others place in it; a slot of menu entries or without entries is left out. */
export function createSlotGroups(source: SlotGroupsSource) {
  return (pluginId: string): readonly SlotGroup[] => {
    const entries = readSlotEntries(source.slots.value) ?? []
    return (source.contributions.slotPoints.value ?? [])
      .filter((point) => point.pluginId === pluginId)
      .map((point) => ({
        slot: point.slot,
        title: point.title,
        contributors: sortSlotEntries(
          entries.filter((one) => one.slot === point.slot && one.view),
        ).flatMap((entry) => {
          const key = getEntryKey(entry)
          return key
            ? [{ key, label: entry.title || prettify(entry.view ?? ''), pluginId: entry.pluginId }]
            : []
        }),
      }))
      .filter((group) => group.contributors.length > 0)
  }
}
