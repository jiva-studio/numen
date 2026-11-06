/**
 * The menu of a tab: the entries of the slot `tabStrip.context` over the tab and its file, with the
 * commands of the window over a tab among them.
 */
import { describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import {
  SLOT,
  type SlotCommand,
  type SlotHost,
  type SlotItem,
  type CommandTarget as RunTarget,
} from '@numen/plugin-host'
import { getTabFileTarget, useTabMenu } from './useTabMenu'

const ATLAS = 'org.example.atlas'
const Pin = defineComponent({ name: 'PinIcon', setup: () => () => h('svg') })

const command = (
  pluginId: string,
  id: string,
  title: string,
  more: Partial<SlotCommand> = {},
): SlotCommand => ({
  kind: 'command',
  key: `${pluginId}/command/${id}`,
  pluginId,
  command: pluginId === 'app' ? `app:${id}` : `${pluginId}:${id}`,
  title,
  shortcut: '',
  hasItems: false,
  ...more,
})

const DETACH = command('app', 'tab.detach', 'Detach tab')
const MOVE = command('app', 'tab.moveToWindow', 'Move to window', { hasItems: true })
const SHOW = command(ATLAS, 'graph.showNote', 'Show note in plex', { icon: Pin })
const ENTRIES: readonly SlotItem[] = [DETACH, MOVE, { kind: 'separator', key: 'separator/1' }, SHOW]

const TARGETS: Record<string, RunTarget> = {
  reef: { path: 'Reef.md', title: 'Reef', isFolder: false, others: [], tab: 'reef' },
  plex: { path: '', title: '', isFolder: false, others: [], tab: 'plex' },
}

/** Slots of a window that hold the entries in the menu of a tab, for every tab but plex. */
const createSlots = (entries: readonly SlotItem[]): SlotHost => ({
  getEntries: (slot, target) =>
    slot === SLOT.tabStripContext && target?.tab !== 'plex' ? entries : [],
  watchEntries: () => ({ dispose: () => undefined }),
})

/** The commands of a window, recording each command run, its target and its argument. */
const createCommands = () => {
  const runs: { command: string; target: RunTarget | undefined; argument?: string }[] = []
  return {
    run: async (id: string, target?: RunTarget, argument?: string) => {
      runs.push(
        argument === undefined ? { command: id, target } : { command: id, target, argument },
      )
    },
    listItems: vi.fn(async (_id: string, _target?: RunTarget) => [
      { id: 'window-a', title: 'Atlas' },
      { id: 'window-b', title: 'Tide log' },
    ]),
    getRuns: () => [...runs],
  }
}

const setup = (entries: readonly SlotItem[] = ENTRIES) => {
  const slots = createSlots(entries)
  const getEntries = vi.spyOn(slots, 'getEntries')
  const commands = createCommands()
  const onHighlight = vi.fn()
  const menu = useTabMenu({
    slots,
    runCommand: commands.run,
    listItems: commands.listItems,
    getTabTarget: (tab) => TARGETS[tab]!,
    onHighlight,
  })
  return { menu, commands, getEntries, onHighlight }
}

const settle = () => new Promise((isDone) => setTimeout(isDone, 0))

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

  it('is the tab and the file it shows, or the note where it shows no file', () => {
    expect(getTabFileTarget(windowTabs, 'book')).toStrictEqual({
      path: 'books/Atlas.pdf',
      title: 'Atlas',
      isFolder: false,
      others: [],
      tab: 'book',
    })
    expect(getTabFileTarget(windowTabs, 'note').path).toBe('Reef.md')
  })

  it('is the tab alone for a tab the window does not hold', () => {
    expect(getTabFileTarget(windowTabs, 'none')).toStrictEqual({
      path: '',
      title: '',
      isFolder: false,
      others: [],
      tab: 'none',
    })
  })
})

describe('the menu of a tab', () => {
  it('is drawn nowhere until it is requested on a tab', () => {
    expect(setup().menu.menu.value).toBeNull()
  })

  it('draws the entries of the slot over the tab, and no row of its own', () => {
    const { menu, getEntries } = setup()

    menu.open('reef', { x: 4, y: 8 })

    expect(menu.menu.value?.tab).toBe('reef')
    expect(menu.menu.value?.at).toStrictEqual({ x: 4, y: 8 })
    expect(menu.menu.value?.items.map((one) => one.id)).toStrictEqual([
      DETACH.key,
      MOVE.key,
      SHOW.key,
    ])
    expect(getEntries).toHaveBeenCalledWith(SLOT.tabStripContext, TARGETS.reef)
  })

  it('stays closed on a tab the slot holds no entry for', () => {
    const { menu } = setup()

    menu.open('plex', { x: 4, y: 8 })

    expect(menu.menu.value).toBeNull()
  })

  it('runs the command of an entry over the tab and its file, and closes', () => {
    const { menu, commands } = setup()
    menu.open('reef', { x: 4, y: 8 })

    menu.choose('reef', DETACH.key)

    expect(commands.getRuns()).toStrictEqual([{ command: DETACH.command, target: TARGETS.reef }])
    expect(menu.menu.value).toBeNull()
  })

  it('runs nothing for a menu since closed, or one requested on another tab', () => {
    const { menu, commands } = setup()
    menu.open('reef', { x: 4, y: 8 })

    menu.choose('plex', DETACH.key)
    menu.dismiss()
    menu.choose('reef', DETACH.key)

    expect(commands.getRuns()).toStrictEqual([])
  })

  it('draws an entry with the icon of its command', () => {
    const { menu } = setup()
    menu.open('reef', { x: 4, y: 8 })

    expect(menu.getIcon(SHOW.key)).toBe(Pin)
    expect(menu.getIcon(DETACH.key)).toBeNull()
  })

  it('draws a view of the slot as a row holding its component, and runs nothing for it', () => {
    const view: SlotItem = {
      kind: 'view',
      key: `${ATLAS}/view/tide`,
      pluginId: ATLAS,
      view: 'tide',
      component: Pin,
    }
    const { menu, commands } = setup([view])
    menu.open('reef', { x: 4, y: 8 })

    expect(menu.menu.value?.items.at(-1)).toMatchObject({ id: view.key, component: Pin })
    menu.choose('reef', view.key)
    expect(commands.getRuns()).toStrictEqual([])
  })
})

describe('the note type of the file of a tab', () => {
  const setupTyped = (readNoteType: (path: string) => Promise<string>) => {
    const slots = createSlots(ENTRIES)
    const getEntries = vi.spyOn(slots, 'getEntries')
    const commands = createCommands()
    const menu = useTabMenu({
      slots,
      runCommand: commands.run,
      listItems: commands.listItems,
      getTabTarget: (tab) => TARGETS[tab]!,
      readNoteType,
    })
    return { menu, getEntries, commands }
  }

  it('is read first, and the slot gives the entries over the typed target', async () => {
    const { menu, getEntries } = setupTyped(async () => 'link')

    menu.open('reef', { x: 4, y: 8 })
    await settle()

    expect(getEntries).toHaveBeenCalledWith(SLOT.tabStripContext, {
      ...TARGETS.reef,
      noteType: 'link',
    })
    expect(menu.menu.value?.tab).toBe('reef')
  })

  it('goes with the target of a command run from the menu', async () => {
    const { menu, commands } = setupTyped(async () => 'link')
    menu.open('reef', { x: 4, y: 8 })
    await settle()

    menu.choose('reef', DETACH.key)

    expect(commands.getRuns()).toStrictEqual([
      { command: DETACH.command, target: { ...TARGETS.reef, noteType: 'link' } },
    ])
  })

  it('is left out where it cannot be read', async () => {
    const { menu, getEntries } = setupTyped(async () => {
      throw new Error('the vault is gone')
    })

    menu.open('reef', { x: 4, y: 8 })
    await settle()

    expect(getEntries).toHaveBeenCalledWith(SLOT.tabStripContext, TARGETS.reef)
  })

  it('opens nothing where another menu was requested while it was read', async () => {
    let finish: (type: string) => void = () => undefined
    const { menu } = setupTyped(
      () =>
        new Promise<string>((done) => {
          finish = done
        }),
    )

    menu.open('reef', { x: 4, y: 8 })
    menu.dismiss()
    finish('link')
    await settle()

    expect(menu.menu.value).toBeNull()
  })
})

describe('an entry of the menu of a tab that offers items', () => {
  const getItemRows = (menu: ReturnType<typeof setup>['menu']) =>
    menu.menu.value?.items
      .find((one) => one.id === MOVE.key)
      ?.children?.map((one) => ({
        id: one.id,
        text: one.text,
        isDisabled: one.isDisabled ?? false,
      }))

  it('asks for its items over the tab when the menu opens, and shows a loading row until then', () => {
    const { menu, commands } = setup()

    menu.open('reef', { x: 4, y: 8 })

    expect(commands.listItems).toHaveBeenCalledWith(MOVE.command, TARGETS.reef)
    expect(getItemRows(menu)?.map((one) => one.isDisabled)).toStrictEqual([true])
  })

  it('is a submenu of its items once they are read', async () => {
    const { menu } = setup()
    menu.open('reef', { x: 4, y: 8 })

    await settle()

    expect(getItemRows(menu)?.map((one) => one.text)).toStrictEqual(['Atlas', 'Tide log'])
  })

  it('runs the command over the tab with the id of the argument picked', async () => {
    const { menu, commands } = setup()
    menu.open('reef', { x: 4, y: 8 })
    await settle()

    menu.choose('reef', getItemRows(menu)![1]!.id)

    expect(commands.getRuns()).toStrictEqual([
      { command: MOVE.command, target: TARGETS.reef, argument: 'window-b' },
    ])
  })

  it('runs nothing for the entry itself', async () => {
    const { menu, commands } = setup()
    menu.open('reef', { x: 4, y: 8 })
    await settle()

    menu.choose('reef', MOVE.key)

    expect(commands.getRuns()).toStrictEqual([])
  })

  it('highlights the window of a move argument under the pointer, and clears the highlight after', async () => {
    const { menu, onHighlight } = setup()
    menu.open('reef', { x: 4, y: 8 })
    await settle()

    menu.highlight(getItemRows(menu)![0]!.id)
    menu.highlight(SHOW.key)

    expect(onHighlight.mock.calls).toStrictEqual([
      ['window-a', true],
      ['window-a', false],
    ])
  })
})
