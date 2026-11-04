/**
 * The ids and titles the editor's own commands are offered under.
 */
import { describe, expect, it } from 'vitest'
import { createCommands } from './table'
import { WORDS as words } from '@/shared/words'

const OWNED = /^app:[a-z][a-zA-Z0-9]*\.[a-z][a-zA-Z0-9]*$/

const TITLES: Readonly<Record<string, string>> = {
  'app:note.open': 'Open note',
  'app:note.openBeside': 'Open note beside',
  'app:note.create': 'Create note',
  'app:note.createChild': 'Create child note',
  'app:note.createParent': 'Create parent note',
  'app:note.createJump': 'Create jump note',
  'app:note.rename': 'Rename note',
  'app:note.trash': 'Move note to trash',
  'app:note.delete': 'Delete note permanently',
  'app:note.copyPath': 'Copy note path',
  'app:note.goToStart': 'Go to start note',
  'app:note.goTo': 'Go to note',
  'app:note.chooseTitleSync': 'Choose title and file name sync',
  'app:tab.close': 'Close tab',
  'app:vault.search': 'Search vault',
  'app:settings.open': 'Open settings',
  'app:appearance.chooseTheme': 'Change theme',
  'app:appearance.chooseMode': 'Choose light or dark mode',
  'app:appearance.chooseInterfaceSize': 'Set interface size',
  'app:appearance.chooseTextSize': 'Set reading font size',
  'app:performance.chooseProfile': 'Set performance profile',
  'app:vault.open': 'Open vault',
  'app:vault.create': 'Create vault',
  'app:vault.rename': 'Rename vault',
  'app:vault.remove': 'Remove vault from list',
  'app:vault.erase': 'Erase vault',
}

describe('the commands of the editor', () => {
  const commands = createCommands(words)

  it('are filed under the owner app and an object and an action', () => {
    for (const one of commands) expect(one.id, one.id).toMatch(OWNED)
  })

  it('are offered under the titles that say what each does', () => {
    const titleById = Object.fromEntries(commands.map((one) => [one.id, one.text]))

    expect(titleById).toStrictEqual(TITLES)
  })
})
