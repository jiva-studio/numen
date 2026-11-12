/** The commands a slot draws, found through what the host knows of its own commands. */
import { describe, expect, it, vi } from 'vitest'
import { SLOT } from '@numen/plugin-sdk'
import { createMockManifest } from '@numen/core-client/testing'
import { createPluginHost } from './host'
import type { CommandDeclaration, SlotCommandFinder } from './types'

const LEDGER: CommandDeclaration = {
  pluginId: 'org.example.ledger',
  id: 'ledger.open',
  title: 'Open ledger',
}

const create = (finder: SlotCommandFinder = () => undefined) => {
  const createSlotCommandFinder = vi.fn(() => finder)
  const host = createPluginHost({
    target: 'desktop.editor',
    registerKind: vi.fn(() => vi.fn()),
    registerExtensionReader: vi.fn(() => vi.fn()),
    openTab: vi.fn(async () => 'tab-1'),
    getCommandDeclarations: () => [LEDGER],
    getSlotEntries: () => [
      { pluginId: LEDGER.pluginId, slot: SLOT.titleBarCommands, command: 'ledger.open' },
    ],
    createSlotCommandFinder,
  })
  return { host, createSlotCommandFinder }
}

describe('the finder of the commands a slot draws', () => {
  it('is made once with the commands the host lists and the commands that offer items', () => {
    const { host, createSlotCommandFinder } = create()
    host.lifecycle
      .createContext(
        LEDGER.pluginId,
        createMockManifest(LEDGER.pluginId, { commands: ['ledger.open'] }),
      )
      .register.command({ id: 'ledger.open', handler: vi.fn(), listItems: async () => [] })

    expect(createSlotCommandFinder).toHaveBeenCalledOnce()
    const [own] = createSlotCommandFinder.mock.calls[0] as unknown as Parameters<
      NonNullable<Parameters<typeof createPluginHost>[0]['createSlotCommandFinder']>
    >
    expect(own.getPluginCommands().map((one) => one.id)).toStrictEqual([
      'org.example.ledger:ledger.open',
    ])
    expect(own.hasPluginItems('org.example.ledger:ledger.open')).toBe(true)
    expect(own.hasPluginItems('org.example.ledger:ledger.close')).toBe(false)
  })

  it('draws the entries of a slot the finder knows', () => {
    const { host } = create(() => ({ title: 'Open ledger' }))

    expect(host.slots.getEntries(SLOT.titleBarCommands)).toMatchObject([{ kind: 'command' }])
  })
})
