/**
 * The commands offered over the note in front.
 */
import { COMMAND } from '@/shared/commandIds'
import { createRowKeys } from '../chords/chords'
import { isNoteOpen } from '../commands/availability'
import type { Command } from '../../types'
import type { Words } from '../../words'

export const createNoteCommands = (words: Words, agent: string): readonly Command[] => [
  {
    id: COMMAND.openNote,
    text: words.read,
    group: 'note',
    isAvailable: isNoteOpen,
    also: COMMAND.openNoteBeside,
  },
  { id: COMMAND.openNoteBeside, text: words.beside, group: 'note', isAvailable: isNoteOpen },
  {
    id: COMMAND.createChildNote,
    text: words.child,
    ...createRowKeys(COMMAND.createChildNote, agent),
    group: 'note',
    needs: 'naming',
    isAvailable: isNoteOpen,
  },
  {
    id: COMMAND.createParentNote,
    text: words.parent,
    group: 'note',
    needs: 'naming',
    isAvailable: isNoteOpen,
  },
  {
    id: COMMAND.createJumpNote,
    text: words.jump,
    group: 'note',
    needs: 'naming',
    isAvailable: isNoteOpen,
  },
  {
    id: COMMAND.renameNote,
    text: words.title,
    group: 'note',
    needs: 'naming',
    isAvailable: isNoteOpen,
    getFieldText: (target) => target.title,
  },
  // The note moves to the vault's .trash folder, and restoring it is a move.
  // Destroy is the command that requires confirmation.
  {
    id: COMMAND.trashNote,
    text: words.remove,
    group: 'note',
    isAvailable: isNoteOpen,
    also: COMMAND.deleteNote,
  },
  {
    id: COMMAND.deleteNote,
    text: words.destroy,
    group: 'note',
    needs: 'exactly',
    isAvailable: isNoteOpen,
    warns: { action: words.destroys, detail: words.forever, back: words.typeBack },
  },
  { id: COMMAND.copyNotePath, text: words.copy, group: 'note', isAvailable: isNoteOpen },
]
