/**
 * The input a command of the palette asks of the person is named by a noun for what is
 * asked, in the field `input`, and the steps that follow it use the same nouns.
 */
import { describe, expect, it } from 'vitest'
import { WORDS as words } from '@/shared/words'
import { createCommands } from './commands'

const commands = createCommands(words)

const getIdsAsking = (kind: string): string[] =>
  commands.filter((one) => one.input === kind).map((one) => one.id)

describe('the input a command of the palette asks for', () => {
  it('is held in the field input, and no command holds a field needs', () => {
    expect(commands.filter((one) => 'needs' in one)).toStrictEqual([])
    expect(commands.filter((one) => one.input !== undefined).length).toBeGreaterThan(0)
  })

  it('is a name for the commands that create or rename', () => {
    expect(getIdsAsking('name')).toStrictEqual([
      'app:note.create',
      'app:note.createChild',
      'app:note.createParent',
      'app:note.createJump',
      'app:note.rename',
      'app:vault.rename',
    ])
  })

  it('is a note for the command that goes to one', () => {
    expect(getIdsAsking('note')).toStrictEqual(['app:note.goTo'])
  })

  it('is a vault for the commands that act on one', () => {
    expect(getIdsAsking('vault')).toStrictEqual([
      'app:vault.open',
      'app:vault.remove',
      'app:vault.erase',
    ])
  })

  it('is a choice for the commands that set how the window is drawn and works', () => {
    expect(getIdsAsking('choice')).toStrictEqual([
      'app:appearance.chooseTheme',
      'app:appearance.chooseMode',
      'app:appearance.chooseInterfaceSize',
      'app:appearance.chooseTextSize',
      'app:note.chooseTitleSync',
      'app:performance.chooseProfile',
    ])
  })

  it('is the name typed again for the command that deletes a note', () => {
    expect(getIdsAsking('retypedName')).toStrictEqual(['app:note.delete'])
  })

  it('is a confirmation after the vault, for the command that removes one', () => {
    const remove = commands.find((one) => one.id === 'app:vault.remove')

    expect([remove?.input, remove?.next]).toStrictEqual(['vault', 'confirmation'])
  })

  it('is the name typed again after the vault, for the command that erases one', () => {
    const erase = commands.find((one) => one.id === 'app:vault.erase')

    expect([erase?.input, erase?.next]).toStrictEqual(['vault', 'retypedName'])
  })

  it('uses no gerund for any step it asks', () => {
    const kinds = new Set(commands.flatMap((one) => [one.input, one.next]))
    kinds.delete(undefined)

    expect([...kinds].sort()).toStrictEqual(
      ['choice', 'confirmation', 'name', 'note', 'retypedName', 'vault'].sort(),
    )
  })
})
