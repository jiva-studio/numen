/**
 * The commands offered over the note in front.
 */
import { ARGUMENT, COMMAND, ROW } from '@/shared/commandIds'
import { createRowKeys } from '../chords/chords'
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
    values: { [ARGUMENT.showing]: 'split' },
    text: words.beside,
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
  // Destroy is the command that requires confirmation.
  {
    id: COMMAND.removeNote,
    text: words.remove,
    group: 'note',
    ...withWhen('target.isNote'),
    also: ROW.destroyNote,
  },
  {
    id: ROW.destroyNote,
    command: COMMAND.removeNote,
    values: { [ARGUMENT.removal]: 'permanent' },
    text: words.destroy,
    group: 'note',
    input: 'retypedName',
    ...withWhen('target.isNote'),
    warns: { action: words.destroys, detail: words.forever, back: words.typeBack },
  },
  { id: COMMAND.copyNotePath, text: words.copy, group: 'note', ...withWhen('target.isNote') },
]
