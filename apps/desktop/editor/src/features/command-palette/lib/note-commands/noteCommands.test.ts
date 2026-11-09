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
      'app:file.open': undefined,
      'app:file.open#beside': undefined,
      'app:note.createChild': 'name',
      'app:note.createParent': 'name',
      'app:note.createJump': 'name',
      'app:note.rename': 'name',
      'app:file.remove': undefined,
      'app:file.remove#permanent': 'retypedName',
      'app:file.copyPath': undefined,
    })
  })
})

describe('the rows that run one command with an argument', () => {
  const commands = createNoteCommands(words, 'linux')
  const find = (id: string) => commands.find((one) => one.id === id)

  it('open the note in the current group or a split by an argument of one command', () => {
    expect(find('app:file.open')).toMatchObject({ also: 'app:file.open#beside' })
    expect(find('app:file.open#beside')).toMatchObject({
      command: 'app:file.open',
      values: { placement: 'split' },
    })
  })

  it('remove the note to the trash or permanently by an argument of one command', () => {
    expect(find('app:file.remove')).toMatchObject({ also: 'app:file.remove#permanent' })
    expect(find('app:file.remove#permanent')).toMatchObject({
      command: 'app:file.remove',
      values: { removal: 'permanent' },
      input: 'retypedName',
    })
  })

  it('hold no command of the old ids', () => {
    const ids = commands.map((one) => one.command ?? one.id)

    expect(ids).not.toContain('app:note.open')
    expect(ids).not.toContain('app:note.openBeside')
    expect(ids).not.toContain('app:note.trash')
    expect(ids).not.toContain('app:note.delete')
  })
})
