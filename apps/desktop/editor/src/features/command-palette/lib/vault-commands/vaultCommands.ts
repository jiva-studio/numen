/**
 * The commands offered over the vaults an installation holds.
 */
import { COMMAND, ROW } from '@/shared/commandIds'
import { createRowKeys } from '../keys/keys'
import { withWhen } from '../commands/availability'
import type { Command } from '../../types'
import type { Words } from '../../words'

export const createVaultCommands = (words: Words, agent: string): readonly Command[] => [
  {
    id: ROW.goToNote,
    command: COMMAND.openFile,
    text: words.goto,
    ...createRowKeys(ROW.goToNote, agent),
    group: 'vault',
    input: 'note',
    ...withWhen('window.isVaultReady'),
  },
  {
    id: COMMAND.openVault,
    text: words.openVault,
    group: 'vault',
    input: 'vault',
    ...withWhen(''),
  },
  {
    id: COMMAND.createVault,
    text: words.newVault,
    ...createRowKeys(COMMAND.createVault, agent),
    group: 'vault',
    ...withWhen(''),
  },
  {
    id: COMMAND.renameVault,
    text: words.renameVault,
    group: 'vault',
    input: 'name',
    ...withWhen('window.isVaultOpen'),
    getFieldText: (target) => target.vault.name,
  },
  {
    id: COMMAND.removeVault,
    text: words.removeVault,
    group: 'vault',
    input: 'vault',
    next: 'confirmation',
    ...withWhen(''),
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
    ...withWhen(''),
    warns: { action: words.erases, detail: words.binned, back: words.typeVaultBack },
  },
]
