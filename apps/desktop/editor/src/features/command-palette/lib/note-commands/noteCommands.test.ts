/**
 * The commands offered over the note in front.
 */
import { describe, expect, it } from 'vitest'
import { WORDS as words } from '@/shared/words'
import { createNoteCommands } from './noteCommands'

describe('the commands over a note', () => {
  it('are all filed with the note commands', () => {
    const commands = createNoteCommands(words, 'linux')

    expect(commands.map((one) => one.group)).toStrictEqual(commands.map(() => 'note'))
  })

  it('ask for a name where they take one', () => {
    const commands = createNoteCommands(words, 'linux')
    const needs = Object.fromEntries(commands.map((one) => [one.id, one.needs]))

    expect(needs).toStrictEqual({
      'app:note.open': undefined,
      'app:note.open-beside': undefined,
      'app:note.create-child': 'naming',
      'app:note.create-parent': 'naming',
      'app:note.create-jump': 'naming',
      'app:note.rename': 'naming',
      'app:note.trash': undefined,
      'app:note.delete': 'exactly',
      'app:note.copy-path': undefined,
    })
  })
})
