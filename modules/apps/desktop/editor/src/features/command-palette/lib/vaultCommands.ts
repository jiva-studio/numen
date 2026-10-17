/**
 * The commands offered over the vaults an installation holds.
 */
import { createRowKeys } from './chords'
import { isAlwaysOffered, isVaultOpen } from './offered'
import type { Command } from '../types'
import type { Words } from '../words'

export const createVaultCommands = (words: Words, agent: string): readonly Command[] => [
  { id: 'first', text: words.first, group: 'vault', isOffered: (at) => at.isReady },
  {
    id: 'goto',
    text: words.goto,
    ...createRowKeys('goto', agent),
    group: 'vault',
    needs: 'picking',
    isOffered: (at) => at.isReady,
  },
  {
    id: 'openVault',
    text: words.openVault,
    group: 'vault',
    needs: 'vaults',
    isOffered: isAlwaysOffered,
  },
  {
    id: 'newVault',
    text: words.newVault,
    ...createRowKeys('newVault', agent),
    group: 'vault',
    isOffered: isAlwaysOffered,
  },
  {
    id: 'renameVault',
    text: words.renameVault,
    group: 'vault',
    needs: 'naming',
    isOffered: isVaultOpen,
    getFieldText: (at) => at.vault.name,
  },
  {
    id: 'removeVault',
    text: words.removeVault,
    group: 'vault',
    needs: 'vaults',
    next: 'asking',
    isOffered: isAlwaysOffered,
    answers: {
      keeps: words.keepsVault,
      kept: words.kept,
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
    isOffered: isAlwaysOffered,
    warns: { action: words.erases, detail: words.binned, back: words.typeVaultBack },
  },
]
