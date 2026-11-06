/**
 * The menu of a tab: the rows the window offers for the tab, then the entries of the slot
 * `tabStrip.context` over the file of the tab.
 */
import { describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import {
  SLOT,
  type SlotCommand,
  type SlotHost,
  type SlotItem,
  type SlotTarget,
} from '@numen/plugin-host'
import type { TabLinksState } from '@/features/tab-linking'
import { getTabFileTarget, useTabMenu } from './useTabMenu'

const ATLAS = 'org.example.atlas'
const Pin = defineComponent({ name: 'PinIcon', setup: () => () => h('svg') })

const SHOW: SlotCommand = {
  kind: 'command',
  key: `${ATLAS}/command/graph.showNote`,
  pluginId: ATLAS,
  command: `${ATLAS}:graph.showNote`,
  title: 'Show note in plex',
  icon: Pin,
  shortcut: '',
}
const COPY: SlotCommand = {
  kind: 'command',
  key: 'app/command/note.copyPath',
  pluginId: 'app',
  command: 'app:note.copyPath',
  title: 'Copy path',
  shortcut: '',
}
const ENTRIES: readonly SlotItem[] = [SHOW, { kind: 'separator', key: 'separator/1' }, COPY]

const TARGETS: Record<string, SlotTarget> = {
  reef: { path: 'Reef.md', title: 'Reef', isFolder: false, others: [] },
  plex: { path: '', title: '', isFolder: false, others: [] },
}

const tabLinks = {
  tabs: [],
  windows: [],
  getGroupOf: () => null,
  getLinkedTargets: () => [],
  linkTabs: vi.fn(),
  unlinkTab: vi.fn(),
} as unknown as TabLinksState

/** Slots of a window that hold the entries in the menu of a tab, and record the commands run. */
const createSlots = (entries: readonly SlotItem[]) => {
  const runs: { command: string; target: SlotTarget | undefined }[] = []
  const slots: SlotHost & { getRuns(): typeof runs } = {
    getEntries: (slot) => (slot === SLOT.tabStripContext ? entries : []),
    watchEntries: () => ({ dispose: () => undefined }),
    runCommand: async (command, target) => {
      runs.push({ command, target })
    },
    getRuns: () => [...runs],
  }
  return slots
}

const setup = (options: { entries?: readonly SlotItem[]; canDetach?: boolean } = {}) => {
  const slots = createSlots(options.entries ?? ENTRIES)
  const getEntries = vi.spyOn(slots, 'getEntries')
  const onDetach = vi.fn()
  const menu = useTabMenu({
    link: {
      tabLinks,
      getTabs: () => [
        { id: 'reef', title: 'Reef' },
        { id: 'plex', title: 'Plex' },
      ],
      getKind: () => 'note',
      isLinkable: () => false,
      ...(options.canDetach === false ? {} : { onDetach }),
    },
    slots,
    getTabTarget: (tab) => TARGETS[tab]!,
    getTabIcon: () => null,
    getKindIcon: () => null,
  })
  return { menu, slots, getEntries, onDetach }
}

describe('the target the entries of the menu of a tab run over', () => {
  const windowTabs = {
    tabs: { value: [] },
    getTab: (id: string) =>
      id === 'none'
        ? undefined
        : {
            state: id,
            kind: {
              getTarget: (state: unknown) =>
                state === 'book'
                  ? { path: '', file: 'books/Atlas.pdf', title: 'Atlas' }
                  : { path: 'Reef.md', file: '', title: 'Reef' },
            },
          },
  } as never

  it('is the file the tab shows, or the note where it shows no file', () => {
    expect(getTabFileTarget(windowTabs, 'book')).toStrictEqual({
      path: 'books/Atlas.pdf',
      title: 'Atlas',
      isFolder: false,
      others: [],
    })
    expect(getTabFileTarget(windowTabs, 'note').path).toBe('Reef.md')
  })

  it('is nothing for a tab the window does not hold', () => {
    expect(getTabFileTarget(windowTabs, 'none')).toStrictEqual({
      path: '',
      title: '',
      isFolder: false,
      others: [],
    })
  })
})

describe('the menu of a tab', () => {
  it('is drawn nowhere until it is requested on a tab', () => {
    const { menu } = setup()

    expect(menu.menu.value).toBeNull()
  })

  it('draws the rows of the window for the tab, then the entries of the slot', () => {
    const { menu } = setup()

    menu.open('reef', { x: 4, y: 8 })

    expect(menu.menu.value?.tab).toBe('reef')
    expect(menu.menu.value?.at).toStrictEqual({ x: 4, y: 8 })
    expect(menu.menu.value?.items.map((one) => one.id)).toStrictEqual([
      '__detach__',
      SHOW.key,
      COPY.key,
    ])
  })

  it('divides the groups of the slot from the rows of the window and from one another', () => {
    const { menu } = setup()

    menu.open('reef', { x: 4, y: 8 })
    const groups = menu.menu.value?.items.map((one) => one.group) ?? []

    expect(new Set(groups).size).toBe(3)
  })

  it('asks the slot for its entries over the file of the tab', () => {
    const { menu, getEntries } = setup()

    menu.open('reef', { x: 4, y: 8 })
    expect(menu.menu.value).not.toBeNull()

    expect(getEntries).toHaveBeenCalledWith(SLOT.tabStripContext, TARGETS.reef)
  })

  it('runs the command of an entry over the file of the tab, and closes', async () => {
    const { menu, slots } = setup()
    menu.open('reef', { x: 4, y: 8 })

    menu.choose('reef', SHOW.key)

    expect(slots.getRuns()).toStrictEqual([{ command: SHOW.command, target: TARGETS.reef }])
    expect(menu.menu.value).toBeNull()
  })

  it('detaches the tab on the row of the window that detaches it', async () => {
    const { menu, slots, onDetach } = setup()
    menu.open('reef', { x: 4, y: 8 })

    menu.choose('reef', '__detach__')

    expect(onDetach).toHaveBeenCalledWith('reef', { x: 4, y: 8 })
    expect(slots.getRuns()).toStrictEqual([])
  })

  it('opens on a tab that only the slot has rows for', () => {
    const { menu } = setup({ canDetach: false })

    menu.open('reef', { x: 4, y: 8 })

    expect(menu.menu.value?.items.map((one) => one.id)).toStrictEqual([SHOW.key, COPY.key])
  })

  it('stays closed on a tab nothing can be done to', () => {
    const { menu } = setup({ canDetach: false, entries: [] })

    menu.open('plex', { x: 4, y: 8 })

    expect(menu.menu.value).toBeNull()
  })

  it('draws an entry of the slot with the icon of its command', () => {
    const { menu } = setup()
    menu.open('reef', { x: 4, y: 8 })

    expect(menu.getIcon(SHOW.key)).toBe(Pin)
    expect(menu.getIcon(COPY.key)).toBeNull()
  })

  it('links, unlinks and moves the tab on the rows of the window that do so', async () => {
    const linkTabs = vi.fn()
    const unlinkTab = vi.fn()
    const onMove = vi.fn()
    const onHighlight = vi.fn()
    const group = { id: 'g', colorIndex: 0, tabs: ['reef', 'plex'], path: '', title: '' }
    const menu = useTabMenu({
      link: {
        tabLinks: {
          ...tabLinks,
          windows: [
            { id: 'window-a', isMain: true, activeTitle: 'Atlas', activeKind: 'note', tabCount: 1 },
          ],
          getGroupOf: (tab: string) => (tab === 'reef' ? group : null),
          linkTabs,
          unlinkTab,
        } as unknown as TabLinksState,
        getTabs: () => [
          { id: 'reef', title: 'Reef' },
          { id: 'plex', title: 'Plex' },
        ],
        getKind: () => 'note',
        isLinkable: () => true,
        onMove,
        onHighlight,
        windowId: 'window-b',
      },
      getTabTarget: (tab) => TARGETS[tab]!,
      getTabIcon: () => null,
      getKindIcon: (kind) => (kind === 'note' ? Pin : null),
    })

    menu.open('reef', { x: 4, y: 8 })
    expect(menu.getIcon('__move__:window-a')).toBe(Pin)
    menu.highlight('__move__:window-a')
    expect(onHighlight).toHaveBeenCalledWith('window-a', true)
    menu.choose('reef', '__move__:window-a')
    menu.open('reef', { x: 4, y: 8 })
    menu.choose('reef', '__unlink__')
    menu.open('reef', { x: 4, y: 8 })
    menu.choose('reef', 'plex')

    expect(onMove).toHaveBeenCalledWith('reef', 'window-a')
    expect(unlinkTab).toHaveBeenCalledWith('reef')
    expect(linkTabs).toHaveBeenCalledWith('reef', 'plex')
  })

  it('draws a view of the slot as a row holding its component, and runs nothing for it', () => {
    const view: SlotItem = {
      kind: 'view',
      key: `${ATLAS}/view/tide`,
      pluginId: ATLAS,
      view: 'tide',
      component: Pin,
    }
    const { menu, slots } = setup({ entries: [view] })
    menu.open('reef', { x: 4, y: 8 })

    expect(menu.menu.value?.items.at(-1)).toMatchObject({ id: view.key, component: Pin })
    menu.choose('reef', view.key)
    expect(slots.getRuns()).toStrictEqual([])
  })

  it('closes on request, and runs nothing', () => {
    const { menu, slots } = setup()
    menu.open('reef', { x: 4, y: 8 })

    menu.dismiss()

    expect(menu.menu.value).toBeNull()
    expect(slots.getRuns()).toStrictEqual([])
  })
})
