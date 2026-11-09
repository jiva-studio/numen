/**
 * The commands offered over the note in front.
 */
import { ARGUMENT, COMMAND, ROW } from '@/shared/commandIds'
import { createRowKeys } from '../keys/keys'
import { withWhen } from '../commands/availability'
import type { Command } from '../../types'
import type { Words } from '../../words'

export const createNoteCommands = (words: Words, agent: string): readonly Command[] => [
  {
    id: COMMAND.openFile,
    text: words.read,
    group: 'note',
    ...withWhen('target.isNote'),
    also: ROW.openNoteBeside,
  },
  {
    id: ROW.openNoteBeside,
    command: COMMAND.openFile,
    values: { [ARGUMENT.placement]: 'split' },
    text: words.beside,
    group: 'note',
    ...withWhen('target.isNote'),
  },
  // The slots draw this command with the kind of place each entry names; the palette offers its rows.
  {
    id: COMMAND.revealFile,
    text: words.reveal,
    group: 'note',
    when: 'target.isNote',
    isAvailable: () => false,
  },
  {
    id: ROW.revealInPlex,
    command: COMMAND.revealFile,
    values: { [ARGUMENT.kind]: 'plex' },
    text: words.revealInPlex,
    group: 'note',
    ...withWhen('target.isNote'),
  },
  {
    id: ROW.revealInFiles,
    command: COMMAND.revealFile,
    values: { [ARGUMENT.kind]: 'files' },
    text: words.revealInFiles,
    group: 'note',
    ...withWhen('target.isNote'),
  },
  {
    id: COMMAND.createChildNote,
    text: words.child,
    ...createRowKeys(COMMAND.createChildNote, agent),
    group: 'note',
    input: 'name',
    ...withWhen('target.isNote'),
  },
  {
    id: COMMAND.createParentNote,
    text: words.parent,
    group: 'note',
    input: 'name',
    ...withWhen('target.isNote'),
  },
  {
    id: COMMAND.createJumpNote,
    text: words.jump,
    group: 'note',
    input: 'name',
    ...withWhen('target.isNote'),
  },
  {
    id: COMMAND.renameNote,
    text: words.title,
    group: 'note',
    input: 'name',
    ...withWhen('target.isNote'),
    getFieldText: (target) => target.title,
  },
  // The note moves to the vault's .trash folder, and restoring it is a move.
  {
    id: COMMAND.removeFile,
    text: words.remove,
    group: 'note',
    ...withWhen('target.isNote'),
  },
  { id: COMMAND.copyFilePath, text: words.copy, group: 'note', ...withWhen('target.isNote') },
]
