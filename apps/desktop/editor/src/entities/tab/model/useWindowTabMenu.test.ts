/**
 * The menu of a tab of the window: the file a tab shows is what the entries of its menu run over,
 * and a command of the menu that fails is reported.
 */
import { describe, expect, it, vi } from 'vitest'
import { SLOT, type SlotCommand, type SlotHost, type SlotItem } from '@numen/plugin-loader'
import { useWindowTabMenu } from './useWindowTabMenu'

const DETACH: SlotCommand = {
  kind: 'command',
  key: 'app/command/tab.detach',
  pluginId: 'app',
  command: 'app:tab.detach',
  title: 'Detach tab',
  shortcut: '',
  hasItems: false,
}

const createSlots = (entries: readonly SlotItem[]): SlotHost => ({
  getEntries: () => entries,
  watchEntries: () => ({ dispose: () => undefined }),
})

const settle = () => new Promise((isDone) => setTimeout(isDone, 0))

describe('the target the entries of the menu of a tab run over', () => {
  const getTab = (id: string) => {
    if (id === 'none') return undefined
    if (id === 'no-address') return { state: id, kind: {} }
    if (id === 'no-target') return { state: id, kind: { address: {} } }
    return {
      state: id,
      kind: {
        address: {
          getTarget: (state: string) =>
            state === 'book'
              ? { path: '', file: 'books/Atlas.pdf', title: 'Atlas' }
              : { path: 'Reef.md', file: '', title: 'Reef' },
        },
      },
    }
  }
  const windowTabs = { tabs: { tabs: { value: [] }, getTab } } as never

  const openOver = async (tab: string) => {
    const slots = createSlots([DETACH])
    const getEntries = vi.spyOn(slots, 'getEntries')
    const menu = useWindowTabMenu({ moveCommand: 'app:tab.moveToWindow', windowTabs, slots })
    menu.open(tab, { x: 4, y: 8 })
    await settle()
    return getEntries
  }

  it('is the tab and the file it shows, or the note where it shows no file', async () => {
    expect(await openOver('book')).toHaveBeenCalledWith(SLOT.tabStripContext, {
      path: 'books/Atlas.pdf',
      title: 'Atlas',
      isFolder: false,
      others: [],
      tab: 'book',
    })
    expect((await openOver('note')).mock.calls[0]?.[1]?.path).toBe('Reef.md')
  })

  it('is the tab alone for a tab the window does not hold or that reports no target', async () => {
    for (const id of ['none', 'no-address', 'no-target']) {
      expect(await openOver(id)).toHaveBeenCalledWith(SLOT.tabStripContext, {
        path: '',
        title: '',
        isFolder: false,
        others: [],
        tab: id,
      })
    }
  })

  it('lists items, highlights move targets, and reads note kinds when configured', async () => {
    const moveEntry: SlotCommand = {
      ...DETACH,
      key: 'app/command/tab.moveToWindow',
      command: 'app:tab.moveToWindow',
      hasItems: true,
    }
    const onHighlight = vi.fn()
    const readNoteKind = vi.fn(async () => ({ noteType: 'sample' }))
    const listItems = vi.fn(async () => [{ id: 'win-2', title: 'Second Window' }])
    const menu = useWindowTabMenu({
      moveCommand: 'app:tab.moveToWindow',
      windowTabs,
      slots: createSlots([moveEntry]),
      commands: {
        getIds: () => [],
        watch: () => ({ dispose: () => undefined }),
        listItems,
        run: async () => '',
      },
      onHighlight,
      readNoteKind,
    })

    menu.open('note', { x: 4, y: 8 })
    await settle()
    expect(readNoteKind).toHaveBeenCalledWith('Reef.md')
    expect(listItems).toHaveBeenCalledOnce()
    const childId = menu.menu.value?.items[0]?.children?.[0]?.id ?? ''
    menu.highlight(childId)
    expect(onHighlight).toHaveBeenCalledWith('win-2', true)

    const menuWithoutCommands = useWindowTabMenu({
      moveCommand: 'app:tab.moveToWindow',
      windowTabs,
      slots: createSlots([moveEntry]),
    })
    menuWithoutCommands.open('note', { x: 4, y: 8 })
    await settle()
    expect(menuWithoutCommands.menu.value?.items[0]?.children).toBeUndefined()
  })
})

describe('a command of the menu of a tab of the window that fails', () => {
  it('is reported once, with the error of its run', async () => {
    const reportError = vi.fn()
    const failure = new Error('the window is gone')
    const menu = useWindowTabMenu({
      moveCommand: 'app:tab.moveToWindow',
      windowTabs: { tabs: { tabs: { value: [] }, getTab: () => undefined } } as never,
      slots: createSlots([DETACH]),
      commands: {
        getIds: () => [],
        watch: () => ({ dispose: () => undefined }),
        listItems: async () => [],
        run: async () => {
          throw failure
        },
      },
      reportError,
    })
    menu.open('reef', { x: 4, y: 8 })
    await settle()

    menu.choose('reef', DETACH.key)
    await settle()

    expect(reportError).toHaveBeenCalledExactlyOnceWith(failure)
  })
})
