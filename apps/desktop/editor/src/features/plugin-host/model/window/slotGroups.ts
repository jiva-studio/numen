import type { Ref } from 'vue'
import {
  getEntryKey,
  SHARED_COMMANDS,
  sortSlotEntries,
  type DeclaredSlotEntry,
} from '@numen/plugin-loader'
import type { Command } from '@/entities/command'
import type { DeclaredCommand, DeclaredSlot } from '@/entities/plugin'
import type { DeclaredSlotPoint } from '@numen/plugin-loader'
import { readSlotEntries } from './pluginSlots'

/** One entry a slot holds, under the key a stored layout names it by. */
interface SlotContributor {
  readonly key: string
  readonly label: string
  readonly pluginId: string
  readonly group: string
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

const prettify = (command: string): string => {
  const words = (command.split(':').pop() ?? command)
    .replace(/([a-z])([A-Z])/g, '$1 $2')
    .replace(/[.-]/g, ' ')
    .toLowerCase()
  return words.charAt(0).toUpperCase() + words.slice(1)
}

/** What the commands of the window and of the plugins are called where a menu shows them. */
interface CommandTitles {
  readonly declared: readonly DeclaredCommand[]
  readonly app: readonly Pick<Command, 'id' | 'text'>[]
}

function getCommandTitle(command: string, titles: CommandTitles): string {
  const declared = titles.declared.find(
    (one) => command === one.id || command === `${one.pluginId}:${one.id}`,
  )
  return (
    declared?.title ||
    titles.app.find((one) => one.id === command)?.text ||
    SHARED_COMMANDS.find((one) => one.id === command)?.title ||
    prettify(command)
  )
}

function getLabel(entry: DeclaredSlotEntry, titles: CommandTitles): string {
  if (entry.title) return entry.title
  if (entry.command) return getCommandTitle(entry.command, titles)
  return prettify(entry.view || entry.submenu || '')
}

/** The slots a plugin draws, each with the entries other plugins and the application place in it. */
export function createSlotGroups(
  source: SlotGroupsSource,
  getAppCommands: () => readonly Pick<Command, 'id' | 'text'>[],
) {
  return (pluginId: string): readonly SlotGroup[] => {
    const entries = readSlotEntries(source.slots.value) ?? []
    const titles = { declared: source.commands.value ?? [], app: getAppCommands() }
    return (source.contributions.slotPoints.value ?? [])
      .filter((point) => point.pluginId === pluginId)
      .map((point) => ({
        slot: point.slot,
        title: point.title,
        contributors: sortSlotEntries(entries.filter((one) => one.slot === point.slot)).flatMap(
          (entry) => {
            const key = getEntryKey(entry)
            return key
              ? [
                  {
                    key,
                    label: getLabel(entry, titles),
                    pluginId: entry.pluginId,
                    group: entry.group ?? '',
                  },
                ]
              : []
          },
        ),
      }))
  }
}
