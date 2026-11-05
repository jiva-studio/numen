/**
 * The commands offered over the vaults an installation holds.
 */
import { COMMAND } from '@/shared/commandIds'
import { createRowKeys } from '../chords/chords'
import { isAlwaysAvailable, isVaultOpen } from '../commands/availability'
import type { Command } from '../../types'
import type { Words } from '../../words'

export const createVaultCommands = (words: Words, agent: string): readonly Command[] => [
  {
    id: COMMAND.goToNote,
    text: words.goto,
    ...createRowKeys(COMMAND.goToNote, agent),
    group: 'vault',
    input: 'note',
    isAvailable: (target) => target.isReady,
  },
  {
    id: COMMAND.openVault,
    text: words.openVault,
    group: 'vault',
    input: 'vault',
    isAvailable: isAlwaysAvailable,
  },
  {
    id: COMMAND.createVault,
    text: words.newVault,
    ...createRowKeys(COMMAND.createVault, agent),
    group: 'vault',
    isAvailable: isAlwaysAvailable,
  },
  {
    id: COMMAND.renameVault,
    text: words.renameVault,
    group: 'vault',
    input: 'name',
    isAvailable: isVaultOpen,
    getFieldText: (target) => target.vault.name,
  },
  {
    id: COMMAND.removeVault,
    text: words.removeVault,
    group: 'vault',
    input: 'vault',
    next: 'confirmation',
    isAvailable: isAlwaysAvailable,
    answers: {
      keeps: words.keepsVault,
      retainedNotice: words.retainedNotice,
      action: words.removeVaultAction,
      detail: words.stays,
    },
  },
  {
    id: COMMAND.eraseVault,
    text: words.eraseVault,
    group: 'vault',
    input: 'vault',
    next: 'retypedName',
    isAvailable: isAlwaysAvailable,
    warns: { action: words.erases, detail: words.binned, back: words.typeVaultBack },
  },
]
