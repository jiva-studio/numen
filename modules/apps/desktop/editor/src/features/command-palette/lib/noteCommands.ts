/**
 * The commands offered over the note in front.
 */
import { keysOf } from './chords'
import { isOnNote } from './offered'
import type { Command } from '../types'
import type { Words } from '../words'

export const noteCommandsOf = (words: Words, agent: string): readonly Command[] => [
  { id: 'read', text: words.read, group: 'note', isOffered: isOnNote, also: 'beside' },
  { id: 'beside', text: words.beside, group: 'note', isOffered: isOnNote },
  {
    id: 'travel',
    text: words.travel,
    ...keysOf('travel', agent),
    group: 'note',
    isOffered: isOnNote,
  },
  {
    id: 'child',
    text: words.child,
    ...keysOf('child', agent),
    group: 'note',
    needs: 'naming',
    isOffered: isOnNote,
  },
  { id: 'parent', text: words.parent, group: 'note', needs: 'naming', isOffered: isOnNote },
  { id: 'jump', text: words.jump, group: 'note', needs: 'naming', isOffered: isOnNote },
  {
    id: 'title',
    text: words.title,
    group: 'note',
    needs: 'naming',
    isOffered: isOnNote,
    getFieldText: (at) => at.title,
  },
  // The note goes to the vault's .trash folder, and putting it back is a move.
  // Destroy is the one that asks.
  { id: 'remove', text: words.remove, group: 'note', isOffered: isOnNote, also: 'destroy' },
  {
    id: 'destroy',
    text: words.destroy,
    group: 'note',
    needs: 'exactly',
    isOffered: isOnNote,
    warns: { action: words.destroys, then: words.forever, back: words.typeBack },
  },
  { id: 'ask', text: words.ask, group: 'note', isOffered: isOnNote },
  { id: 'copy', text: words.copy, group: 'note', isOffered: isOnNote },
  { id: 'reveal', text: words.reveal, group: 'note', isOffered: isOnNote },
  { id: 'preset', text: words.preset, group: 'note', isOffered: isOnNote },
]
