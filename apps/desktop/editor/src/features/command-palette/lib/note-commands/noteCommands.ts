/**
 * The commands offered over the note in front.
 */
import { createRowKeys } from '../chords/chords'
import { isNoteOpen } from '../commands/availability'
import type { Command } from '../../types'
import type { Words } from '../../words'

export const createNoteCommands = (words: Words, agent: string): readonly Command[] => [
  { id: 'read', text: words.read, group: 'note', isOffered: isNoteOpen, also: 'beside' },
  { id: 'beside', text: words.beside, group: 'note', isOffered: isNoteOpen },
  {
    id: 'travel',
    text: words.travel,
    ...createRowKeys('travel', agent),
    group: 'note',
    isOffered: (target, runs) => isNoteOpen(target) && runs.canRun('plex'),
  },
  {
    id: 'child',
    text: words.child,
    ...createRowKeys('child', agent),
    group: 'note',
    needs: 'naming',
    isOffered: isNoteOpen,
  },
  { id: 'parent', text: words.parent, group: 'note', needs: 'naming', isOffered: isNoteOpen },
  { id: 'jump', text: words.jump, group: 'note', needs: 'naming', isOffered: isNoteOpen },
  {
    id: 'title',
    text: words.title,
    group: 'note',
    needs: 'naming',
    isOffered: isNoteOpen,
    getFieldText: (target) => target.title,
  },
  // The note goes to the vault's .trash folder, and putting it back is a move.
  // Destroy is the one that asks.
  { id: 'remove', text: words.remove, group: 'note', isOffered: isNoteOpen, also: 'destroy' },
  {
    id: 'destroy',
    text: words.destroy,
    group: 'note',
    needs: 'exactly',
    isOffered: isNoteOpen,
    warns: { action: words.destroys, detail: words.forever, back: words.typeBack },
  },
  {
    id: 'ask',
    text: words.ask,
    group: 'note',
    isOffered: (target, runs) => isNoteOpen(target) && runs.canRun('agent'),
  },
  { id: 'copy', text: words.copy, group: 'note', isOffered: isNoteOpen },
  {
    id: 'reveal',
    text: words.reveal,
    group: 'note',
    isOffered: (target, runs) => isNoteOpen(target) && runs.canRun('files'),
  },
]
