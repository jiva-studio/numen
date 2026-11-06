/**
 * What the title of the window offers to switch to: the vaults of the installation as the
 * welcome screen lists them, the one in the window marked, and a last row for adding a folder.
 */
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import type { CommandTarget, VaultRef } from '@/features/command-palette'
import type { VaultList } from '@/entities/vault'
import { COMMAND } from '@/shared/commandIds'
import { useVaultSwitcher } from './useVaultSwitcher'

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
  const openVault = vi.fn<(vault: VaultRef) => void>()
  const bar = useVaultSwitcher({
    vaultList,
    words: { add: 'Create vault' },
    runCommand,
    openVault,
    getTarget: () => target,
  })
  return { bar, vaultList, runCommand, openVault }
}

describe('the vaults of the title', () => {
  it('list the vaults in the order of the vault list and then the row that adds a folder', () => {
    const { bar } = setup()

    expect(bar.vaults.value.map((one) => [one.id, one.name])).toStrictEqual([
      ['orchard', 'Orchard notes'],
      ['greenhouse', 'Greenhouse log'],
      ['cellar', 'Cellar book'],
      [COMMAND.createVault, 'Create vault'],
    ])
  })

  it('mark the vault of the window as current and no other row', () => {
    const { bar } = setup()

    expect(bar.vaults.value.filter((one) => one.isCurrent).map((one) => one.id)).toStrictEqual([
      'greenhouse',
    ])
  })

  it('follow the vault list when it changes', () => {
    const { bar, vaultList } = setup()

    vaultList.value = { vaults: [vault('cellar', 'Cellar book')], showing: 'cellar' }

    expect(bar.vaults.value.map((one) => one.id)).toStrictEqual(['cellar', COMMAND.createVault])
    expect(bar.vaults.value[0]?.isCurrent).toBe(true)
  })

  it('are none while the window shows no vault', () => {
    const { bar } = setup({ ...LIST, showing: '' })

    expect(bar.vaults.value).toStrictEqual([])
  })
})

describe('choosing in the title', () => {
  it('opens a vault of the list as the welcome screen does, with the id and the name of the vault', () => {
    const { bar, openVault, runCommand } = setup()

    bar.switchVault('cellar')

    expect(openVault.mock.calls).toStrictEqual([[{ id: 'cellar', name: 'Cellar book' }]])
    expect(runCommand).not.toHaveBeenCalled()
  })

  it('runs the command that adds a folder from the last row', () => {
    const { bar, openVault, runCommand } = setup()

    bar.switchVault(COMMAND.createVault)

    expect(runCommand.mock.calls).toStrictEqual([[COMMAND.createVault, target]])
    expect(openVault).not.toHaveBeenCalled()
  })

  it('does nothing for an id the list does not hold', () => {
    const { bar, openVault, runCommand } = setup()

    bar.switchVault('lost')

    expect(openVault).not.toHaveBeenCalled()
    expect(runCommand).not.toHaveBeenCalled()
  })
})
