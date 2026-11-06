/**
 * The vaults the window gives its plugins: the vaults of the installation as the welcome screen
 * lists them and a last row for adding a folder, the one the window shows, and the switch to another.
 */
import { describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import type { VaultsState } from '@numen/plugin-host'
import type { CommandTarget, VaultRef } from '@/features/command-palette'
import type { VaultList } from '@/entities/vault'
import { COMMAND } from '@/shared/commandIds'
import { createWindowVaults } from './windowVaults'

const vault = (id: string, name: string) => ({
  id,
  name,
  path: `/vaults/${name}`,
  isMissing: false,
})

const LIST: VaultList = {
  vaults: [
    vault('orchard', 'Orchard notes'),
    vault('greenhouse', 'Greenhouse log'),
    vault('cellar', 'Cellar book'),
  ],
  showing: 'greenhouse',
}

const target = { vault: { id: 'greenhouse', name: 'Greenhouse log' } } as unknown as CommandTarget

function setup(list: VaultList = LIST) {
  const vaultList = ref(list)
  const runCommand = vi.fn<(id: string, target: CommandTarget) => void>()
  const openVault = vi.fn<(vault: VaultRef) => Promise<void>>(async () => undefined)
  const vaults = createWindowVaults({
    vaultList,
    words: { add: 'Create vault' },
    runCommand,
    openVault,
    getTarget: () => target,
  })
  return { vaults, vaultList, runCommand, openVault }
}

describe('the vaults of the window', () => {
  it('list the vaults in the order of the vault list and then the row that adds a folder', () => {
    const { vaults } = setup()

    expect(vaults.getState().vaults).toStrictEqual([
      { id: 'orchard', name: 'Orchard notes' },
      { id: 'greenhouse', name: 'Greenhouse log' },
      { id: 'cellar', name: 'Cellar book' },
      { id: COMMAND.createVault, name: 'Create vault' },
    ])
  })

  it('name the vault the window shows as the current one', () => {
    expect(setup().vaults.getState().current).toStrictEqual({
      id: 'greenhouse',
      name: 'Greenhouse log',
    })
  })

  it('name no current vault while the window shows none', () => {
    expect(setup({ ...LIST, showing: '' }).vaults.getState().current).toBeNull()
  })

  it('call a watcher at once and again when the vault list changes, until it is disposed', async () => {
    const { vaults, vaultList } = setup()
    const states: VaultsState[] = []
    const watch = vaults.watch((state) => states.push(state))

    vaultList.value = { vaults: [vault('cellar', 'Cellar book')], showing: 'cellar' }
    await nextTick()
    watch.dispose()
    vaultList.value = LIST
    await nextTick()

    expect(states.map((one) => one.current?.id)).toStrictEqual(['greenhouse', 'cellar'])
    expect(states[1]?.vaults.map((one) => one.id)).toStrictEqual(['cellar', COMMAND.createVault])
  })
})

describe('switching the vault of the window', () => {
  it('opens a vault of the list as the welcome screen does, with the id and the name of the vault', async () => {
    const { vaults, openVault, runCommand } = setup()

    await vaults.switchTo('cellar')

    expect(openVault.mock.calls).toStrictEqual([[{ id: 'cellar', name: 'Cellar book' }]])
    expect(runCommand).not.toHaveBeenCalled()
  })

  it('runs the command that adds a folder for the last row', async () => {
    const { vaults, openVault, runCommand } = setup()

    await vaults.switchTo(COMMAND.createVault)

    expect(runCommand.mock.calls).toStrictEqual([[COMMAND.createVault, target]])
    expect(openVault).not.toHaveBeenCalled()
  })

  it('does nothing for an id the list does not hold', async () => {
    const { vaults, openVault, runCommand } = setup()

    await vaults.switchTo('lost')

    expect(openVault).not.toHaveBeenCalled()
    expect(runCommand).not.toHaveBeenCalled()
  })
})
