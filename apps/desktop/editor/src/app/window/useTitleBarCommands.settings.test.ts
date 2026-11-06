/** The settings button of the application in the title bar, over the real slots. */
import { describe, expect, it, vi } from 'vitest'
import { shallowRef } from 'vue'
import { createSlotRegistry } from '@numen/plugin-host'
import type { DeclaredSlot } from '@/entities/plugin'
import { createCommands, type CommandTarget } from '@/features/command-palette'
import { COMMAND } from '@/shared/commandIds'
import { WORDS } from '@/shared/words'
import { createSlotCommandFinder, readSlotEntries } from '../commands/pluginSlots'
import { useTitleBarCommands } from './useTitleBarCommands'

const LINUX = 'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/605.1.15'
const TIDE = 'org.example.tide'
const target = { tab: 't' } as unknown as CommandTarget

function setup(isVaultOpen: boolean) {
  const placements: DeclaredSlot[] = [
    { pluginId: TIDE, slot: 'titleBar.commands', command: 'tide.open', group: 'navigate' },
    { pluginId: TIDE, slot: 'titleBar.commands', command: 'tide.add', group: 'new' },
  ]
  const slots = createSlotRegistry({
    getEntries: () => readSlotEntries(placements),
    findCommand: createSlotCommandFinder({
      getPluginCommands: () => [
        { id: `${TIDE}:tide.open`, title: 'Open tide', icon: 'waves' },
        { id: `${TIDE}:tide.add`, title: 'Add tide', icon: 'plus' },
      ],
      getDeclaredCommands: () => [],
      getEditorCommands: () => createCommands(WORDS),
      getWhenValues: () => ({}),
    }),
    findView: () => undefined,
  })
  const open = shallowRef(isVaultOpen)
  const runCommand = vi.fn<(id: string, target: CommandTarget) => void>()
  const bar = useTitleBarCommands({
    slots,
    isVaultOpen: () => open.value,
    runCommand,
    getTarget: () => target,
    agent: LINUX,
  })
  return { bar, open, runCommand }
}

describe('the title bar with the settings button', () => {
  it('has the entries of the plugins and then the settings while a vault is open', () => {
    const { bar } = setup(true)

    expect(bar.commands.value.map((one) => one.id)).toEqual([
      `${TIDE}:tide.open`,
      `${TIDE}:tide.add`,
      'app:settings.open',
    ])
  })

  it('has no button, the settings included, while no vault is open', () => {
    const { bar, open } = setup(false)

    expect(bar.commands.value).toEqual([])
    open.value = true
    expect(bar.commands.value.at(-1)?.id).toBe('app:settings.open')
  })

  it('carries the title of the command in the hover text of the settings button', () => {
    const { bar } = setup(true)

    expect(bar.commands.value.at(-1)).toMatchObject({ title: 'Open settings', group: '2' })
  })

  it('runs the settings command as the palette runs it on a press', () => {
    const { bar, runCommand } = setup(true)

    bar.runCommand(COMMAND.openSettings)

    expect(runCommand).toHaveBeenCalledWith(COMMAND.openSettings, target)
  })
})
