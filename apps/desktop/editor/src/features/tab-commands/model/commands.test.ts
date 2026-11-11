/**
 * The commands of the window over a tab: detach it, unlink it, move it to another window and link it
 * with another tab, the last two with one item per window and per tab.
 */
import { describe, expect, it, vi } from 'vitest'
import type { CommandTarget } from '@/shared/model/command'
import type { LinkableTab, TabLinkGroup } from '@/entities/tab'
import { invoke } from '@/testing/invocation'
import { createTabCommands, type TabCommandsDeps } from './commands'

type TabLinks = TabCommandsDeps['tabLinks']

const GROUP: TabLinkGroup = { id: 'g', colorIndex: 0, tabs: ['reef', 'plex'], path: '', title: '' }

const DETACH = 'app:tab.detach'
const UNLINK = 'app:tab.unlink'
const MOVE = 'app:tab.moveToWindow'
const LINK = 'app:tab.link'

const ELSEWHERE: LinkableTab = { id: 'dock', title: 'Dock', kind: 'note', windowId: 'main' }
const UNTITLED: LinkableTab = { id: 'quay', title: '', kind: 'files', windowId: 'main' }

const createLinks = (groups: Record<string, TabLinkGroup> = {}) => ({
  tabs: [ELSEWHERE, UNTITLED],
  windows: [
    { id: 'main', isMain: true, activeTitle: 'Atlas', activeKind: 'note', tabCount: 3 },
    { id: 'here', isMain: false, activeTitle: 'Reef', activeKind: 'note', tabCount: 1 },
    { id: 'side', isMain: false, activeTitle: '', activeKind: '', tabCount: 0 },
  ],
  getGroupOf: (tab: string) => groups[tab] ?? null,
  linkTabs: vi.fn<TabLinks['linkTabs']>(),
  unlinkTab: vi.fn<TabLinks['unlinkTab']>(),
})

const setup = (
  options: {
    groups?: Record<string, TabLinkGroup>
    isAsking?: boolean
  } = {},
) => {
  const tabLinks = createLinks(options.groups)
  const detach = vi.fn()
  const move = vi.fn()
  const set = createTabCommands({
    tabLinks,
    getTabs: () => [
      { id: 'reef', title: 'Reef' },
      { id: 'plex', title: 'Plex' },
      { id: 'chat', title: 'Chat' },
    ],
    getKind: (tab) =>
      tab === 'chat' ? 'agent' : ['dock', 'quay', 'lost'].includes(tab) ? undefined : 'note',
    isLinkable: (tab) => tab !== 'chat',
    windowId: 'here',
    isAsking: () => options.isAsking ?? false,
    detach,
    move,
  })
  return { set, tabLinks, detach, move }
}

const createTabTarget = (tab: string): CommandTarget => ({
  tab,
  kind: 'note',
  path: '',
  title: '',
  file: '',
  vault: { id: 'v', name: 'V' },
  isReady: true,
})

describe('the commands of a tab', () => {
  it('are detaching, unlinking, moving and linking it, the last two listing items', () => {
    const { set } = setup()

    expect(set.commands.map((one) => [one.id, one.text, one.input ?? null])).toStrictEqual([
      [DETACH, 'Detach tab', null],
      [UNLINK, 'Unlink tab', null],
      [MOVE, 'Move to window', 'items'],
      [LINK, 'Link with tab', 'items'],
    ])
    expect(Object.keys(set.handlers).sort()).toStrictEqual([DETACH, LINK, MOVE, UNLINK].sort())
  })

  it('offer moving to every other window, named by its active tab and the tabs besides it', () => {
    const { set } = setup()
    const move = set.commands.find((one) => one.id === MOVE)!

    expect(move.listItems?.(createTabTarget('reef'))).toStrictEqual([
      { id: 'main', title: 'Atlas (+2)' },
      { id: 'side', title: 'Empty window' },
    ])
  })

  it('offer linking with the other linkable tabs of every window, named by title or else by kind', () => {
    const { set } = setup()
    const link = set.commands.find((one) => one.id === LINK)!

    expect(link.listItems?.(createTabTarget('reef'))).toStrictEqual([
      { id: 'plex', title: 'Plex' },
      { id: 'dock', title: 'Dock' },
      { id: 'quay', title: 'files' },
    ])
  })

  it('offer no tab of the group the tab is in, and none from a tab that cannot be linked', async () => {
    const { set } = setup({ groups: { reef: GROUP, plex: GROUP } })
    const link = set.commands.find((one) => one.id === LINK)!

    expect((await link.listItems?.(createTabTarget('reef')))?.map((one) => one.id)).toStrictEqual([
      'dock',
      'quay',
    ])
    expect(await link.listItems?.(createTabTarget('chat'))).toStrictEqual([])
  })

  it('are available over a tab only, unlink over a linked tab, and none while the tab asks', () => {
    const linked = setup({ groups: { reef: GROUP } }).set.commands
    const blocked = setup({ isAsking: true }).set.commands
    const isCommandAvailable = (list: typeof linked, id: string, tab: string) =>
      list.find((one) => one.id === id)!.isAvailable(createTabTarget(tab), {} as never)

    expect(isCommandAvailable(linked, UNLINK, 'reef')).toBe(true)
    expect(isCommandAvailable(linked, UNLINK, 'chat')).toBe(false)
    expect(isCommandAvailable(linked, DETACH, '')).toBe(false)
    expect(isCommandAvailable(blocked, DETACH, 'reef')).toBe(false)
    expect(isCommandAvailable(blocked, MOVE, 'reef')).toBe(false)
    expect(isCommandAvailable(linked, LINK, 'chat')).toBe(false)
  })

  it('run over the tab of the target, with the item picked', async () => {
    const { set, tabLinks, detach, move } = setup({ groups: { reef: GROUP } })
    const target = createTabTarget('reef')

    await set.handlers[DETACH]?.(invoke(DETACH, target))
    await set.handlers[UNLINK]?.(invoke(UNLINK, target))
    await set.handlers[MOVE]?.(invoke(MOVE, target, 'main'))
    await set.handlers[LINK]?.(invoke(LINK, target, 'chat'))

    expect(detach).toHaveBeenCalledWith('reef')
    expect(tabLinks.unlinkTab).toHaveBeenCalledWith('reef')
    expect(move).toHaveBeenCalledWith('reef', 'main')
    expect(tabLinks.linkTabs).toHaveBeenCalledWith('reef', 'chat')
  })

  it('run nothing without a tab, or a move and a link without an item', async () => {
    const { set, tabLinks, detach, move } = setup()

    await set.handlers[DETACH]?.(invoke(DETACH, createTabTarget('')))
    await set.handlers[MOVE]?.(invoke(MOVE, createTabTarget('reef')))
    await set.handlers[LINK]?.(invoke(LINK, createTabTarget('reef')))

    expect(detach).not.toHaveBeenCalled()
    expect(move).not.toHaveBeenCalled()
    expect(tabLinks.linkTabs).not.toHaveBeenCalled()
  })
})
