/**
 * The commands of the window over a tab: detach it, unlink it, move it to another window and link it
 * with another tab, the last two with one item per window and per tab.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { commands, createInvocation, type CommandTarget } from '@/features/command-palette'
import type { TabLinkGroup, TabLinksState } from '@/features/tab-linking'
import { createTabCommands, TAB_COMMAND } from './tabCommands'

const GROUP: TabLinkGroup = { id: 'g', colorIndex: 0, tabs: ['reef', 'plex'], path: '', title: '' }

const createState = (groups: Record<string, TabLinkGroup> = {}) =>
  ({
    tabs: [],
    windows: [
      { id: 'main', isMain: true, activeTitle: 'Atlas', activeKind: 'note', tabCount: 3 },
      { id: 'here', isMain: false, activeTitle: 'Reef', activeKind: 'note', tabCount: 1 },
      { id: 'side', isMain: false, activeTitle: '', activeKind: '', tabCount: 0 },
    ],
    getGroupOf: (tab: string) => groups[tab] ?? null,
    getLinkedTargets: () => [],
    linkTabs: vi.fn(),
    unlinkTab: vi.fn(),
  }) as unknown as TabLinksState & { linkTabs: ReturnType<typeof vi.fn> }

const setup = (options: { groups?: Record<string, TabLinkGroup>; isAsking?: boolean } = {}) => {
  const tabLinks = createState(options.groups)
  const detach = vi.fn()
  const move = vi.fn()
  const tabCommands = createTabCommands({
    tabLinks,
    getTabs: () => [
      { id: 'reef', title: 'Reef' },
      { id: 'plex', title: 'Plex' },
      { id: 'chat', title: 'Chat' },
    ],
    getKind: (tab) => (tab === 'chat' ? 'agent' : 'note'),
    isLinkable: (tab) => tab !== 'chat',
    windowId: 'here',
    isAsking: () => options.isAsking ?? false,
    detach,
    move,
  })
  return { tabCommands, tabLinks, detach, move }
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

afterEach(() => commands.clear())

describe('the commands of a tab', () => {
  it('are detaching, unlinking, moving and linking it, the last two listing items', () => {
    const { tabCommands } = setup()

    expect(tabCommands.commands.map((one) => [one.id, one.text, one.input ?? null])).toStrictEqual([
      [TAB_COMMAND.detach, 'Detach tab', null],
      [TAB_COMMAND.unlink, 'Unlink tab', null],
      [TAB_COMMAND.moveToWindow, 'Move to window', 'items'],
      [TAB_COMMAND.link, 'Link with tab', 'items'],
    ])
  })

  it('offer moving to every other window, named by its tab in front', () => {
    const { tabCommands } = setup()
    const move = tabCommands.commands.find((one) => one.id === TAB_COMMAND.moveToWindow)!

    expect(move.listItems?.(createTabTarget('reef'))).toStrictEqual([
      { id: 'main', title: 'Atlas (+2)' },
      { id: 'side', title: 'Empty window' },
    ])
  })

  it('offer linking with every linkable tab outside the group of the tab', () => {
    const { tabCommands } = setup({ groups: { reef: GROUP, plex: GROUP } })
    const link = tabCommands.commands.find((one) => one.id === TAB_COMMAND.link)!

    expect(link.listItems?.(createTabTarget('reef'))).toStrictEqual([])
    expect(link.listItems?.(createTabTarget('chat'))).toStrictEqual([])
    expect(setup().tabCommands.commands[3]!.listItems?.(createTabTarget('reef'))).toStrictEqual([
      { id: 'plex', title: 'Plex' },
    ])
  })

  it('are available over a tab only, unlink over a linked tab, and none while the tab asks', () => {
    const linked = setup({ groups: { reef: GROUP } }).tabCommands.commands
    const askingTabCommands = setup({ isAsking: true }).tabCommands.commands
    const isCommandAvailable = (list: typeof linked, id: string, tab: string) =>
      list.find((one) => one.id === id)!.isAvailable(createTabTarget(tab), {} as never)

    expect(isCommandAvailable(linked, TAB_COMMAND.unlink, 'reef')).toBe(true)
    expect(isCommandAvailable(linked, TAB_COMMAND.unlink, 'chat')).toBe(false)
    expect(isCommandAvailable(linked, TAB_COMMAND.detach, '')).toBe(false)
    expect(isCommandAvailable(askingTabCommands, TAB_COMMAND.detach, 'reef')).toBe(false)
    expect(isCommandAvailable(askingTabCommands, TAB_COMMAND.moveToWindow, 'reef')).toBe(false)
    expect(isCommandAvailable(linked, TAB_COMMAND.link, 'chat')).toBe(false)
  })

  it('run over the tab of the target, with the argument picked', async () => {
    const { tabCommands, tabLinks, detach, move } = setup({ groups: { reef: GROUP } })
    tabCommands.register()

    await commands.get(TAB_COMMAND.detach)!(
      createInvocation(TAB_COMMAND.detach, createTabTarget('reef')),
      {} as never,
      {} as never,
    )
    await commands.get(TAB_COMMAND.unlink)!(
      createInvocation(TAB_COMMAND.unlink, createTabTarget('reef')),
      {} as never,
      {} as never,
    )
    await commands.get(TAB_COMMAND.moveToWindow)!(
      createInvocation(TAB_COMMAND.moveToWindow, createTabTarget('reef'), 'main'),
      {} as never,
      {} as never,
    )
    await commands.get(TAB_COMMAND.link)!(
      createInvocation(TAB_COMMAND.link, createTabTarget('reef'), 'chat'),
      {} as never,
      {} as never,
    )

    expect(detach).toHaveBeenCalledWith('reef')
    expect(tabLinks.unlinkTab).toHaveBeenCalledWith('reef')
    expect(move).toHaveBeenCalledWith('reef', 'main')
    expect(tabLinks.linkTabs).toHaveBeenCalledWith('reef', 'chat')
  })

  it('run nothing without a tab, or a move and a link without a argument', async () => {
    const { tabCommands, tabLinks, detach, move } = setup()
    const unregister = tabCommands.register()

    await commands.get(TAB_COMMAND.detach)!(
      createInvocation(TAB_COMMAND.detach, createTabTarget('')),
      {} as never,
      {} as never,
    )
    await commands.get(TAB_COMMAND.moveToWindow)!(
      createInvocation(TAB_COMMAND.moveToWindow, createTabTarget('reef')),
      {} as never,
      {} as never,
    )
    await commands.get(TAB_COMMAND.link)!(
      createInvocation(TAB_COMMAND.link, createTabTarget('reef')),
      {} as never,
      {} as never,
    )
    unregister()

    expect(detach).not.toHaveBeenCalled()
    expect(move).not.toHaveBeenCalled()
    expect(tabLinks.linkTabs).not.toHaveBeenCalled()
    expect(commands.get(TAB_COMMAND.detach)).toBeUndefined()
  })
})
