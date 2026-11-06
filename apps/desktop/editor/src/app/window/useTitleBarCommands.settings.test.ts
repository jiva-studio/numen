/** The settings button of the application in the title bar, over the real menus. */
import { describe, expect, it, vi } from 'vitest'
import { markRaw, shallowRef, type Component } from 'vue'
import type { MenuPlacement } from '@numen/plugin-host'
import { createCommands, type CommandTarget } from '@/features/command-palette'
import { COMMAND } from '@/shared/commandIds'
import { WORDS } from '@/shared/words'
import { createPluginMenus } from '../commands/pluginMenus'
import { useTitleBarCommands } from './useTitleBarCommands'

const LINUX = 'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/605.1.15'
const TIDE = 'org.example.tide'
const target = { tab: 't' } as unknown as CommandTarget

function setup(isVaultOpen: boolean) {
  const placements: MenuPlacement[] = [
    { pluginId: TIDE, location: 'titleBar.commands', command: 'tide.open', group: 'navigate' },
    { pluginId: TIDE, location: 'titleBar.commands', command: 'tide.add', group: 'new' },
  ]
  const menus = createPluginMenus({
    getPlacements: () => placements,
    getPluginCommands: () => [
      { id: `${TIDE}:tide.open`, title: 'Open tide', icon: 'waves' },
      { id: `${TIDE}:tide.add`, title: 'Add tide', icon: 'plus' },
    ],
    getDeclaredCommands: () => [],
    getEditorCommands: () => createCommands(WORDS),
  })
  const open = shallowRef(isVaultOpen)
  const runCommand = vi.fn<(id: string, target: CommandTarget) => void>()
  const bar = useTitleBarCommands({
    menus,
    isVaultOpen: () => open.value,
    runCommand,
    getTarget: () => target,
    getIcon: (name): Component => markRaw({ render: () => null, name }),
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

    expect(bar.commands.value.at(-1)).toMatchObject({ title: 'Open settings', group: 'settings' })
  })

  it('runs the settings command as the palette runs it on a press', () => {
    const { bar, runCommand } = setup(true)

    bar.runCommand(COMMAND.openSettings)

    expect(runCommand).toHaveBeenCalledWith(COMMAND.openSettings, target)
  })
})
