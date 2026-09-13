/**
 * The commands offered over the vaults an installation holds.
 */
import { keysOf } from './chords'
import { isOnAnything, isOnVault } from './offered'
import type { Command } from '../types'
import type { Words } from '../words'

export const vaultCommandsOf = (words: Words, agent: string): readonly Command[] => [
  { id: 'first', text: words.first, group: 'vault', isOffered: (at) => at.ready },
  {
    id: 'goto',
    text: words.goto,
    ...keysOf('goto', agent),
    group: 'vault',
    needs: 'picking',
    isOffered: (at) => at.ready,
  },
  { id: 'openVault', text: words.openVault, group: 'vault', needs: 'vaults', isOffered: isOnAnything },
  {
    id: 'newVault',
    text: words.newVault,
    ...keysOf('newVault', agent),
    group: 'vault',
    isOffered: isOnAnything,
  },
  {
    id: 'renameVault',
    text: words.renameVault,
    group: 'vault',
    needs: 'naming',
    isOffered: isOnVault,
    getFieldText: (at) => at.vault.name,
  },
  {
    id: 'forgetVault',
    text: words.forgetVault,
    group: 'vault',
    needs: 'vaults',
    next: 'asking',
    isOffered: isOnAnything,
    answers: {
      keeps: words.keepsVault,
      kept: words.kept,
      action: words.forgets,
      then: words.stays,
    },
  },
  {
    id: 'eraseVault',
    text: words.eraseVault,
    group: 'vault',
    needs: 'vaults',
    next: 'exactly',
    isOffered: isOnAnything,
    warns: { action: words.erases, then: words.binned, back: words.typeVaultBack },
  },
]
