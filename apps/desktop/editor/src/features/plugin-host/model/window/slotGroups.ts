import type { Ref } from 'vue'
import { getEntryKey, sortSlotEntries, type DeclaredSlotEntry } from '@numen/plugin-loader'
import type { DeclaredCommand, DeclaredSlot } from '@/entities/plugin'
import type { DeclaredSlotPoint } from '@numen/plugin-loader'
import { readSlotEntries } from './pluginSlots'

/** One entry a slot holds, under the key a stored layout names it by. */
interface SlotContributor {
  readonly key: string
  readonly label: string
  readonly pluginId: string
}

/** A slot a plugin draws and the entries placed in it, in the order it draws them when nothing is stored. */
interface SlotGroup {
  readonly slot: string
  readonly title: string
  readonly contributors: readonly SlotContributor[]
}

/** What the groups are read from. */
interface SlotGroupsSource {
  readonly slots: Readonly<Ref<readonly DeclaredSlot[] | undefined>>
  readonly commands: Readonly<Ref<readonly DeclaredCommand[] | undefined>>
  readonly contributions: {
    readonly slotPoints: Readonly<Ref<readonly DeclaredSlotPoint[] | undefined>>
  }
}

function getLabel(entry: DeclaredSlotEntry, commands: readonly DeclaredCommand[]): string {
  const command = commands.find(
    (one) => entry.command === one.id || entry.command === `${one.pluginId}:${one.id}`,
  )
  return entry.title || command?.title || entry.command || entry.view || entry.submenu || ''
}

/** The slots a plugin draws, each with the entries other plugins and the application place in it. */
export function createSlotGroups(source: SlotGroupsSource) {
  return (pluginId: string): readonly SlotGroup[] => {
    const entries = readSlotEntries(source.slots.value) ?? []
    const commands = source.commands.value ?? []
    return (source.contributions.slotPoints.value ?? [])
      .filter((point) => point.pluginId === pluginId)
      .map((point) => ({
        slot: point.slot,
        title: point.title,
        contributors: sortSlotEntries(entries.filter((one) => one.slot === point.slot)).flatMap(
          (entry) => {
            const key = getEntryKey(entry)
            return key ? [{ key, label: getLabel(entry, commands), pluginId: entry.pluginId }] : []
          },
        ),
      }))
  }
}
