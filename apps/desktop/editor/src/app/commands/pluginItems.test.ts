/**
 * The pick list of a command with items in the palette, and the commands the palette leaves out
 * because they list no item over what is in front.
 */
import { describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import type { CommandItem } from '@numen/plugin-host'
import type { Command, CommandTarget, PaletteLists, RunSupport } from '@/features/command-palette'
import { createItemLists } from './pluginItems'

const TARGET = {
  tab: '',
  kind: null,
  path: 'tides.md',
  file: 'tides.md',
  title: 'tides',
  vault: { id: 'v', name: 'Harbor' },
  isReady: true,
} as CommandTarget

const WINDOW_LISTS: PaletteLists = { getStepGroups: () => [], previewItem: () => undefined }
const RUNS: RunSupport = { canRun: () => true, cannotRun: () => undefined }

const createCommand = (id: string): Command => ({
  id,
  text: id,
  group: 'note',
  isAvailable: () => true,
})

const settle = async () => {
  for (let round = 0; round < 4; round += 1) await nextTick()
}

function setUp(itemsById: Readonly<Record<string, readonly CommandItem[]>>) {
  const listCommandItems = vi.fn(async (id: string) => itemsById[id] ?? [])
  const host = { hasCommandItems: (id: string) => id in itemsById, listCommandItems }
  const lists = createItemLists(WINDOW_LISTS, host, () => TARGET, RUNS)
  return { ...lists, listCommandItems }
}

describe('the commands with items in the palette', () => {
  it('are left out once their items are read and none is listed', async () => {
    const { removeEmptyCommands } = setUp({
      'tab.move': [{ id: 'w1', title: 'Window 1' }],
      'version.delete': [],
    })
    const listed = [createCommand('tab.move'), createCommand('version.delete'), createCommand('x')]

    removeEmptyCommands(listed)
    await settle()

    expect(removeEmptyCommands(listed).map((one) => one.id)).toStrictEqual(['tab.move', 'x'])
  })

  it('are listed while their items are being read', () => {
    const { removeEmptyCommands } = setUp({ 'version.delete': [] })

    expect(removeEmptyCommands([createCommand('version.delete')]).map((one) => one.id)).toStrictEqual([
      'version.delete',
    ])
  })

  it('read no items of a command that is not available over what is in front', () => {
    const { removeEmptyCommands, listCommandItems } = setUp({ 'version.delete': [] })

    removeEmptyCommands([{ ...createCommand('version.delete'), isAvailable: () => false }])

    expect(listCommandItems).not.toHaveBeenCalled()
  })

  it('are read again each time the palette opens', async () => {
    const { removeEmptyCommands, resetWhenClosed, listCommandItems } = setUp({
      'version.delete': [],
    })
    const isOpen = ref(false)
    resetWhenClosed(isOpen)
    removeEmptyCommands([createCommand('version.delete')])
    await settle()

    isOpen.value = true
    await nextTick()
    removeEmptyCommands([createCommand('version.delete')])

    expect(listCommandItems).toHaveBeenCalledTimes(2)
  })
})
