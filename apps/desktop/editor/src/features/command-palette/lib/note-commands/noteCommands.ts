/**
 * The commands offered over the note in front.
 */
import { createRowKeys } from '../chords/chords'
import { isNoteOpen } from '../commands/availability'
import type { Command } from '../../types'
import type { Words } from '../../words'

export const createNoteCommands = (words: Words, agent: string): readonly Command[] => [
  { id: 'read', text: words.read, group: 'note', isAvailable: isNoteOpen, also: 'beside' },
  { id: 'beside', text: words.beside, group: 'note', isAvailable: isNoteOpen },
  {
    id: 'child',
    text: words.child,
    ...createRowKeys('child', agent),
    group: 'note',
    needs: 'naming',
    isAvailable: isNoteOpen,
  },
  { id: 'parent', text: words.parent, group: 'note', needs: 'naming', isAvailable: isNoteOpen },
  { id: 'jump', text: words.jump, group: 'note', needs: 'naming', isAvailable: isNoteOpen },
  {
    id: 'title',
    text: words.title,
    group: 'note',
    needs: 'naming',
    isAvailable: isNoteOpen,
    getFieldText: (target) => target.title,
  },
  // The note moves to the vault's .trash folder, and restoring it is a move.
  // Destroy is the command that asks for confirmation.
  { id: 'remove', text: words.remove, group: 'note', isAvailable: isNoteOpen, also: 'destroy' },
  {
    id: 'destroy',
    text: words.destroy,
    group: 'note',
    needs: 'exactly',
    isAvailable: isNoteOpen,
    warns: { action: words.destroys, detail: words.forever, back: words.typeBack },
  },
  { id: 'copy', text: words.copy, group: 'note', isAvailable: isNoteOpen },
]
