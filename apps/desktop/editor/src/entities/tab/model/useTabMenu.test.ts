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
} from '@numen/plugin-loader'
import { useTabMenu, type NoteKind } from './useTabMenu'

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

const createSlots = (entries: readonly SlotItem[]): SlotHost => ({
  getEntries: (slot, target) =>
    slot === SLOT.tabStripContext && target?.tab !== 'plex' ? entries : [],
  watchEntries: () => ({ dispose: () => undefined }),
})

const createCommands = () => {
  const runs: {
    command: string
    target: RunTarget | undefined
    itemId?: string
    values?: Readonly<Record<string, string>>
  }[] = []
  return {
    run: async (
      id: string,
      target?: RunTarget,
      itemId?: string,
      values?: Readonly<Record<string, string>>,
    ) => {
      runs.push({
        command: id,
        target,
        ...(itemId === undefined ? {} : { itemId }),
        ...(values === undefined ? {} : { values }),
      })
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
    moveCommand: 'app:tab.moveToWindow',
    slots,
    runCommand: commands.run,
    listItems: commands.listItems,
    getTabTarget: (tab) => TARGETS[tab]!,
    onHighlight,
  })
  return { menu, commands, getEntries, onHighlight }
}

const settle = () => new Promise((isDone) => setTimeout(isDone, 0))

describe('the menu of a tab', () => {
  it('is drawn nowhere until it is requested on a tab', () => {
    expect(setup().menu.menu.value).toBeNull()
  })

  it('draws the entries of the slot over the tab, and no row of its own', async () => {
    const { menu, getEntries } = setup()

    menu.open('reef', { x: 4, y: 8 })
    await settle()

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

  it('runs the command of an entry over the tab and its file, and closes', async () => {
    const { menu, commands } = setup()
    menu.open('reef', { x: 4, y: 8 })
    await settle()

    menu.choose('reef', DETACH.key)

    expect(commands.getRuns()).toStrictEqual([{ command: DETACH.command, target: TARGETS.reef }])
    expect(menu.menu.value).toBeNull()
  })

  it('runs the command of an entry with the values the entry holds', async () => {
    const withPlace = command(ATLAS, 'graph.show', 'Show', { values: { where: 'plex' } })
    const { menu, commands } = setup([withPlace])
    menu.open('reef', { x: 4, y: 8 })
    await settle()

    menu.choose('reef', withPlace.key)

    expect(commands.getRuns()).toStrictEqual([
      { command: withPlace.command, target: TARGETS.reef, values: { where: 'plex' } },
    ])
  })

  it('runs nothing for a menu since closed, or one requested on another tab', async () => {
    const { menu, commands } = setup()
    menu.open('reef', { x: 4, y: 8 })
    await settle()

    menu.choose('plex', DETACH.key)
    menu.dismiss()
    menu.choose('reef', DETACH.key)

    expect(commands.getRuns()).toStrictEqual([])
  })

  it('draws an entry with the icon of its command', async () => {
    const { menu } = setup()
    menu.open('reef', { x: 4, y: 8 })
    await settle()

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

describe('the note type and the properties of the file of a tab', () => {
  const setupTyped = (readNoteKind: (path: string) => Promise<NoteKind>) => {
    const slots = createSlots(ENTRIES)
    const getEntries = vi.spyOn(slots, 'getEntries')
    const commands = createCommands()
    const menu = useTabMenu({
      moveCommand: 'app:tab.moveToWindow',
      slots,
      runCommand: commands.run,
      listItems: commands.listItems,
      getTabTarget: (tab) => TARGETS[tab]!,
      readNoteKind,
    })
    return { menu, getEntries, commands }
  }

  it('is read first, and the slot gives the entries over the typed target', async () => {
    const { menu, getEntries } = setupTyped(async () => ({ noteType: 'link' }))

    menu.open('reef', { x: 4, y: 8 })
    await settle()

    expect(getEntries).toHaveBeenCalledWith(SLOT.tabStripContext, {
      ...TARGETS.reef,
      noteType: 'link',
    })
    expect(menu.menu.value?.tab).toBe('reef')
  })

  it('gives the slot the properties of the note with its type', async () => {
    const { menu, getEntries } = setupTyped(async () => ({
      noteType: 'link',
      properties: { media: 'plugin://md.importer/a.mp4' },
    }))

    menu.open('reef', { x: 4, y: 8 })
    await settle()

    expect(getEntries).toHaveBeenCalledWith(SLOT.tabStripContext, {
      ...TARGETS.reef,
      noteType: 'link',
      properties: { media: 'plugin://md.importer/a.mp4' },
    })
  })

  it('goes with the target of a command run from the menu', async () => {
    const { menu, commands } = setupTyped(async () => ({ noteType: 'link' }))
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
    let finish: (kind: NoteKind) => void = () => undefined
    const { menu } = setupTyped(
      () =>
        new Promise<NoteKind>((done) => {
          finish = done
        }),
    )

    menu.open('reef', { x: 4, y: 8 })
    menu.dismiss()
    finish({ noteType: 'link' })
    await settle()

    expect(menu.menu.value).toBeNull()
  })
})

describe('an open of the menu of a tab cancelled while its note type is read', () => {
  it('opens nothing after Escape', async () => {
    let finish: (kind: NoteKind) => void = () => undefined
    const commands = createCommands()
    const menu = useTabMenu({
      moveCommand: 'app:tab.moveToWindow',
      slots: createSlots(ENTRIES),
      runCommand: commands.run,
      listItems: commands.listItems,
      getTabTarget: (tab) => TARGETS[tab]!,
      readNoteKind: () => new Promise<NoteKind>((done) => (finish = done)),
    })

    menu.open('reef', { x: 4, y: 8 })
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    finish({ noteType: 'link' })
    await settle()
    await settle()

    expect(menu.menu.value).toBeNull()
    expect(commands.listItems).not.toHaveBeenCalled()
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

  it('asks for its items over the tab when the menu opens, and is not drawn until they are read', () => {
    const { menu, commands } = setup()

    menu.open('reef', { x: 4, y: 8 })

    expect(commands.listItems).toHaveBeenCalledWith(MOVE.command, TARGETS.reef)
    expect(menu.menu.value).toBeNull()
  })

  it('is drawn with a loading row once the hold has passed', async () => {
    vi.useFakeTimers()
    const { menu, commands } = setup()
    commands.listItems.mockReturnValue(new Promise(() => undefined))

    menu.open('reef', { x: 4, y: 8 })
    await vi.advanceTimersByTimeAsync(150)
    vi.useRealTimers()

    expect(getItemRows(menu)?.map((one) => one.isDisabled)).toStrictEqual([true])
  })

  it('is not drawn after Escape is pressed while its items are read', async () => {
    const { menu } = setup()

    menu.open('reef', { x: 4, y: 8 })
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await settle()

    expect(menu.menu.value).toBeNull()
  })

  it('is a submenu of its items once they are read', async () => {
    const { menu } = setup()
    menu.open('reef', { x: 4, y: 8 })

    await settle()

    expect(getItemRows(menu)?.map((one) => one.text)).toStrictEqual(['Atlas', 'Tide log'])
  })

  it('runs the command over the tab with the id of the itemId picked', async () => {
    const { menu, commands } = setup()
    menu.open('reef', { x: 4, y: 8 })
    await settle()

    menu.choose('reef', getItemRows(menu)![1]!.id)

    expect(commands.getRuns()).toStrictEqual([
      { command: MOVE.command, target: TARGETS.reef, itemId: 'window-b' },
    ])
  })

  it('runs nothing for the entry itself', async () => {
    const { menu, commands } = setup()
    menu.open('reef', { x: 4, y: 8 })
    await settle()

    menu.choose('reef', MOVE.key)

    expect(commands.getRuns()).toStrictEqual([])
  })

  it('highlights the window of a move itemId under the pointer, and clears the highlight after', async () => {
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
