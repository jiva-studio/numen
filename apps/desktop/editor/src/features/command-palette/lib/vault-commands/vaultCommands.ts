/**
 * The commands offered over the vaults an installation holds.
 */
import { createRowKeys } from '../chords/chords'
import { isAlwaysAvailable, isVaultOpen } from '../commands/availability'
import type { Command } from '../../types'
import type { Words } from '../../words'

export const createVaultCommands = (words: Words, agent: string): readonly Command[] => [
  { id: 'first', text: words.first, group: 'vault', isAvailable: (target) => target.isReady },
  {
    id: 'goto',
    text: words.goto,
    ...createRowKeys('goto', agent),
    group: 'vault',
    needs: 'picking',
    isAvailable: (target) => target.isReady,
  },
  {
    id: 'openVault',
    text: words.openVault,
    group: 'vault',
    needs: 'vaults',
    isAvailable: isAlwaysAvailable,
  },
  {
    id: 'newVault',
    text: words.newVault,
    ...createRowKeys('newVault', agent),
    group: 'vault',
    isAvailable: isAlwaysAvailable,
  },
  {
    id: 'renameVault',
    text: words.renameVault,
    group: 'vault',
    needs: 'naming',
    isAvailable: isVaultOpen,
    getFieldText: (target) => target.vault.name,
  },
  {
    id: 'removeVault',
    text: words.removeVault,
    group: 'vault',
    needs: 'vaults',
    next: 'asking',
    isAvailable: isAlwaysAvailable,
    answers: {
      keeps: words.keepsVault,
      retainedNotice: words.retainedNotice,
      action: words.removeVaultAction,
      detail: words.stays,
    },
  },
  {
    id: 'eraseVault',
    text: words.eraseVault,
    group: 'vault',
    needs: 'vaults',
    next: 'exactly',
    isAvailable: isAlwaysAvailable,
    warns: { action: words.erases, detail: words.binned, back: words.typeVaultBack },
  },
]
