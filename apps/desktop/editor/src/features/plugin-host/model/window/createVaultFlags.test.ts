/**
 * The flags a plugin sets on a vault reach the conditions of the menus of the window: the entries of
 * the menu over a row change with the flag of the vault shown. Every plugin here is invented.
 */
import { describe, expect, it, vi } from 'vitest'
import { computed } from 'vue'
import { createSlotRegistry, createWhenReader } from '@numen/plugin-loader'
import type { CommandTarget } from '@/shared/model/command'
import { createVaultFlags } from './createVaultFlags'
import { createWindowFacts } from './createWindowFacts'
import { createWindowSlots } from './createWindowSlots'

const PLUGIN = 'org.example.mirror'
const ROW = { path: 'Sea', title: 'Sea', isFolder: true, others: [], tab: '' }

function setup() {
  const vault = { id: 'sea' }
  const flags = createVaultFlags()
  const slots = createWindowSlots({
    declaredCommands: {
      slots: computed(() => [
        {
          pluginId: PLUGIN,
          slot: 'fileTree.context',
          command: 'mirror.start',
          when: 'vault.flags.org.example.mirror.isMirrored == false',
        },
        {
          pluginId: PLUGIN,
          slot: 'fileTree.context',
          command: 'mirror.share',
          when: 'vault.flags.org.example.mirror.isMirrored == true',
        },
      ]),
      commands: computed(() => [
        { pluginId: PLUGIN, id: 'mirror.start', title: 'Start', arguments: [] },
        { pluginId: PLUGIN, id: 'mirror.share', title: 'Share', arguments: [] },
      ]),
      contributions: {
        slotPoints: computed(() => [
          { pluginId: 'org.example.tree', slot: 'fileTree.context', title: 'Row menu' },
        ]),
        handlers: computed(() => []),
      },
      whenContext: {
        getContentTypes: () => ({}),
        getConfig: () => ({}),
        whenReader: createWhenReader(),
      },
    },
    facts: createWindowFacts({
      appCommands: [],
      getTarget: () => ({ tab: '', vault, isReady: true }) as CommandTarget,
      isPaletteOpen: () => false,
      getVaultFlags: flags.getFlags,
    }),
    writeIssue: vi.fn(),
  })
  const registry = createSlotRegistry({
    getEntries: () => slots.hostDeps.slots.getSlotEntries(new Map()),
    findCommand: slots.hostDeps.slots.createSlotCommandFinder({
      getPluginCommands: () => [],
      hasPluginItems: () => false,
    }),
    findView: () => undefined,
    when: {
      getValues: slots.hostDeps.slots.getSlotWhenValues,
      reader: slots.hostDeps.commands.whenReader,
    },
  })
  const list = () =>
    registry
      .getEntries('fileTree.context', ROW)
      .flatMap((one) => (one.kind === 'command' ? [one.command] : []))
  return { vault, flags: flags.createHost(PLUGIN), list }
}

describe('the menu of a row of the file tree', () => {
  it('offers the entry of a true flag and not the entry of a false flag', () => {
    const { flags, list } = setup()

    flags.setFlag('sea', 'isMirrored', false)
    expect(list()).toStrictEqual([`${PLUGIN}:mirror.start`])
    flags.setFlag('sea', 'isMirrored', true)
    expect(list()).toStrictEqual([`${PLUGIN}:mirror.share`])
  })

  it('offers neither entry while the plugin has set no flag, and reads the flag of the vault shown', () => {
    const { flags, list } = setup()

    expect(list()).toStrictEqual([])
    flags.setFlag('other', 'isMirrored', true)
    expect(list()).toStrictEqual([])
  })
})
