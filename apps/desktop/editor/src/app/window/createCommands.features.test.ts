/** The commands of the features as the window collects them. */
import { describe, expect, it } from 'vitest'
import type { CommandTarget } from '@/shared/model/command'
import { WORDS } from '@/shared/ui/words'
import { createEmptyWindow } from '@/testing/appCommands'
import { collectFeatureCommands } from './createFeatureCommands'

const NO_WINDOW = createEmptyWindow()

/** The commands the palette carries out itself, with no handler declared by a feature. */
const OWNED_BY_PALETTE = ['app:tab.close', 'app:vault.search', 'app:settings.open']

/** The command that shows a note is drawn by the slots with a kind; the palette offers its rows. */
const DRAWN_BY_SLOTS = ['app:file.reveal']

describe('the commands of the features', () => {
  const collected = collectFeatureCommands(NO_WINDOW, WORDS, 'linux')

  it('are carried out by something, each row either by a handler or through the command it names', () => {
    const missing = collected.commands
      .map((one) => one.command ?? one.id)
      .filter((id) => !DRAWN_BY_SLOTS.includes(id) && !OWNED_BY_PALETTE.includes(id))
      .filter((id) => collected.handlers[id] === undefined)

    expect(missing).toStrictEqual([])
  })

  it('are drawn in the order of their groups: note, file, window, settings, vault', () => {
    const ids = collected.commands.map((one) => one.id)

    expect(ids.indexOf('app:note.create')).toBeLessThan(ids.indexOf('app:folder.create'))
    expect(ids.indexOf('app:folder.create')).toBeLessThan(ids.indexOf('app:tab.close'))
    expect(ids.indexOf('app:tab.close')).toBeLessThan(ids.indexOf('app:vault.search'))
    expect(ids.indexOf('app:settings.open')).toBeLessThan(ids.indexOf('app:file.open#note'))
  })

  it('hold each id once', () => {
    const ids = collected.commands.map((one) => one.id)

    expect(new Set(ids).size).toBe(ids.length)
  })

  it('keep the rows of each group in the order they were drawn before the features declared them', () => {
    const byGroup = (group: string) =>
      collected.commands.filter((one) => one.group === group).map((one) => one.id)

    expect(byGroup('note')).toStrictEqual([
      'app:file.open',
      'app:file.open#beside',
      'app:file.reveal',
      'app:note.createChild',
      'app:note.createParent',
      'app:note.createJump',
      'app:note.rename',
      'app:file.remove',
      'app:file.copyPath',
    ])
    expect(byGroup('window')).toStrictEqual([
      'app:note.create',
      'app:folder.create',
      'app:tab.close',
      'app:vault.search',
      'app:tab.detach',
      'app:tab.unlink',
      'app:tab.moveToWindow',
      'app:tab.link',
    ])
    expect(byGroup('file')).toStrictEqual(['app:file.open#document', 'app:version.delete'])
    expect(byGroup('settings')).toStrictEqual(['app:settings.open'])
    expect(byGroup('vault')).toStrictEqual([
      'app:file.open#note',
      'app:vault.open',
      'app:vault.create',
      'app:vault.rename',
      'app:vault.remove',
      'app:vault.erase',
      'app:trash.empty',
    ])
  })
})

describe('the commands over a tab', () => {
  const target = (tab: string): CommandTarget => ({
    tab,
    kind: 'note',
    path: '',
    title: '',
    file: '',
    vault: { id: 'v', name: 'V' },
    isReady: true,
  })
  const collectWithTabs = () =>
    collectFeatureCommands(
      {
        ...NO_WINDOW,
        tabs: {
          ...NO_WINDOW.tabs,
          tabLinks: {
            ...NO_WINDOW.tabs.tabLinks,
            tabs: [{ id: 'dock', title: 'Dock', kind: 'note', windowId: 'main' }],
            windows: [
              { id: 'main', isMain: true, activeTitle: 'Atlas', activeKind: 'note', tabCount: 3 },
              { id: 'here', isMain: false, activeTitle: 'Reef', activeKind: 'note', tabCount: 1 },
            ],
          },
          windowId: 'here',
          getTabs: () => [
            { id: 'reef', title: 'Reef' },
            { id: 'plex', title: 'Plex' },
            { id: 'chat', title: 'Chat' },
          ],
          getKind: (tab) => (tab === 'chat' ? 'agent' : 'note'),
          isLinkable: (tab) => tab !== 'chat',
        },
      },
      WORDS,
      'linux',
    ).commands

  it('offer linking with the linkable tabs of every window, other than the tab itself', () => {
    const link = collectWithTabs().find((one) => one.id === 'app:tab.link')

    expect(link?.listItems?.(target('reef'))).toStrictEqual([
      { id: 'plex', title: 'Plex' },
      { id: 'dock', title: 'Dock' },
    ])
    expect(link?.listItems?.(target('chat'))).toStrictEqual([])
  })

  it('offer moving to the other windows, each named by its tab in front', () => {
    const move = collectWithTabs().find((one) => one.id === 'app:tab.moveToWindow')

    expect(move?.listItems?.(target('reef'))).toStrictEqual([{ id: 'main', title: 'Atlas (+2)' }])
  })
})
