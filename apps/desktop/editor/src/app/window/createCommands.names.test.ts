/**
 * The ids and titles the editor's own commands are offered under.
 */
import { describe, expect, it } from 'vitest'
import { createAppCommands } from '@/testing/appCommands'

const OWNED = /^app:[a-z][a-zA-Z0-9]*\.[a-z][a-zA-Z0-9]*$/

const TITLES: Readonly<Record<string, string>> = {
  'app:file.open': 'Open note',
  'app:file.open#beside': 'Open note beside',
  'app:file.reveal': 'Show note',
  'app:note.create': 'Create note',
  'app:folder.create': 'Create folder',
  'app:note.createChild': 'Create child note',
  'app:note.createParent': 'Create parent note',
  'app:note.createJump': 'Create jump note',
  'app:note.rename': 'Rename note',
  'app:file.remove': 'Move note to trash',
  'app:file.copyPath': 'Copy note path',
  'app:file.open#note': 'Go to note',
  'app:file.open#document': 'Open the document here',
  'app:tab.close': 'Close tab',
  'app:vault.search': 'Search vault',
  'app:settings.open': 'Open settings',
  'app:vault.open': 'Open vault',
  'app:vault.create': 'Create vault',
  'app:vault.rename': 'Rename vault',
  'app:vault.remove': 'Remove vault from list',
  'app:vault.erase': 'Erase vault',
  'app:trash.empty': 'Empty trash',
  'app:tab.detach': 'Detach tab',
  'app:tab.unlink': 'Unlink tab',
  'app:tab.moveToWindow': 'Move to window',
  'app:tab.link': 'Link with tab',
  'app:version.delete': 'Delete text',
}

describe('the commands of the editor', () => {
  const commands = createAppCommands()

  it('are filed under the owner app and an object and an action', () => {
    for (const one of commands) expect(one.command ?? one.id, one.id).toMatch(OWNED)
  })

  it('offer no way to go to a start note', () => {
    expect(commands.map((one) => one.id)).not.toContain('app:note.goToStart')
    expect(commands.map((one) => one.text)).not.toContain('Go to start note')
  })

  it('are offered under the titles that say what each does', () => {
    const titleById = Object.fromEntries(commands.map((one) => [one.id, one.text]))

    expect(titleById).toStrictEqual(TITLES)
  })
})
