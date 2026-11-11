import { describe, expect, it } from 'vitest'
import type { CommandTarget } from '@/shared/model/command'
import type { DeclaredSlot } from '@/entities/plugin'
import { createEmptyWhenContext } from './whenContext'
import { createSlotPaletteCommands } from './slotPaletteCommands'

const entry = (patch: Partial<DeclaredSlot>): DeclaredSlot => ({
  pluginId: 'org.example.atlas',
  slot: 'palette.note',
  command: 'app:file.reveal',
  values: { kind: 'atlas' },
  title: 'Show in atlas',
  when: 'target.isNote',
  ...patch,
})

const target = (patch: Partial<CommandTarget>): CommandTarget =>
  ({
    isReady: true,
    tab: 'tab-1',
    vault: { id: 'vault-1' },
    path: 'notes/Entropy.md',
    file: 'notes/Entropy.md',
    ...patch,
  }) as CommandTarget

describe('the rows of the slot palette.note', () => {
  it('lists the command of an entry with its values under the title the plugin gives', () => {
    const [row] = createSlotPaletteCommands([entry({})], createEmptyWhenContext())

    expect(row).toMatchObject({
      id: 'app:file.reveal#kind=atlas',
      command: 'app:file.reveal',
      values: { kind: 'atlas' },
      text: 'Show in atlas',
      group: 'note',
    })
  })

  it('lists nothing for the entries of other slots or without a title, and nothing while no plugin is listed', () => {
    const context = createEmptyWhenContext()

    expect(
      createSlotPaletteCommands(
        [entry({ slot: 'palette.searchResult' }), entry({ title: '' })],
        context,
      ),
    ).toStrictEqual([])
    expect(createSlotPaletteCommands(undefined, context)).toStrictEqual([])
  })

  it('is available over a note and not over another file', () => {
    const [row] = createSlotPaletteCommands([entry({})], createEmptyWhenContext())

    expect(row?.isAvailable(target({}), {} as never)).toBe(true)
    expect(row?.isAvailable(target({ path: 'scan.pdf', file: 'scan.pdf' }), {} as never)).toBe(
      false,
    )
  })
})
