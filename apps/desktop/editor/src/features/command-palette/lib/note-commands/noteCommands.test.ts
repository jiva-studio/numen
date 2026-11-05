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
    const inputs = Object.fromEntries(commands.map((one) => [one.id, one.input]))

    expect(inputs).toStrictEqual({
      'app:note.open': undefined,
      'app:note.openBeside': undefined,
      'app:note.createChild': 'name',
      'app:note.createParent': 'name',
      'app:note.createJump': 'name',
      'app:note.rename': 'name',
      'app:note.trash': undefined,
      'app:note.delete': 'retypedName',
      'app:note.copyPath': undefined,
    })
  })
})
