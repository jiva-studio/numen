/**
 * The ids and titles the editor's own commands are offered under.
 */
import { describe, expect, it } from 'vitest'
import { createCommands } from './table'
import { WORDS as words } from '@/shared/words'

const OWNED = /^app:[a-z]+(-[a-z]+)*\.[a-z]+(-[a-z]+)*$/

const TITLES: Readonly<Record<string, string>> = {
  'app:note.open': 'Open note',
  'app:note.open-beside': 'Open note beside',
  'app:note.create': 'Create note',
  'app:note.create-child': 'Create child note',
  'app:note.create-parent': 'Create parent note',
  'app:note.create-jump': 'Create jump note',
  'app:note.rename': 'Rename note',
  'app:note.trash': 'Move note to trash',
  'app:note.delete': 'Delete note permanently',
  'app:note.copy-path': 'Copy note path',
  'app:note.go-to-start': 'Go to start note',
  'app:note.go-to': 'Go to note',
  'app:note.choose-title-sync': 'Choose title and file name sync',
  'app:tab.close': 'Close tab',
  'app:vault.search': 'Search vault',
  'app:settings.open': 'Open settings',
  'app:appearance.choose-theme': 'Change theme',
  'app:appearance.choose-mode': 'Choose light or dark mode',
  'app:appearance.choose-interface-size': 'Set interface size',
  'app:appearance.choose-text-size': 'Set reading font size',
  'app:performance.choose-profile': 'Set performance profile',
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
    const offered = Object.fromEntries(commands.map((one) => [one.id, one.text]))

    expect(offered).toStrictEqual(TITLES)
  })
})
