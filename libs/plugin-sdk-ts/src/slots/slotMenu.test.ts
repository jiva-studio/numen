/**
 * The menu tree of the entries of a slot, and the item lists read for their commands.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { CommandItem } from '../commands/commands'
import type { SlotCommand, SlotItem, SlotSeparator, SlotSubmenu, SlotView } from './slots'
import {
  createSlotMenu,
  MENU_HOLD_MS,
  readCommandItemLists,
  type CommandItemLists,
} from './slotMenu'

const WORDS = { loadingText: 'Loading…' }

const command = (id: string, extra: Partial<SlotCommand> = {}): SlotCommand => ({
  kind: 'command',
  key: `md.numen.test/command/${id}`,
  pluginId: 'md.numen.test',
  command: `md.numen.test:${id}`,
  title: id,
  shortcut: '',
  hasItems: false,
  ...extra,
})

const separator = (index: number): SlotSeparator => ({
  kind: 'separator',
  key: `separator/${index}`,
})

const submenu = (id: string, entries: readonly SlotItem[]): SlotSubmenu => ({
  kind: 'submenu',
  key: `md.numen.test/submenu/${id}`,
  pluginId: 'md.numen.test',
  submenu: id,
  title: id,
  entries,
})

const VIEW: SlotView = {
  kind: 'view',
  key: 'md.numen.test/view/sketch',
  pluginId: 'md.numen.test',
  view: 'sketch',
  component: { name: 'Sketch' },
}

const OPEN = command('open', { icon: 'open-icon' })
const RENAME = command('rename')
const MOVE = command('moveToWindow', { hasItems: true })
const CREATE_TEXT = command('createText', {
  key: 'md.numen.test/command/app:file.createText',
  command: 'app:file.createText',
  title: 'Create text',
  hasItems: true,
})

const lists = (pairs: readonly [SlotCommand, readonly CommandItem[] | null][]): CommandItemLists =>
  new Map(pairs.map(([entry, items]) => [entry.key, items]))

describe('createSlotMenu', () => {
  it('puts the rows between two separators into one group', () => {
    const menu = createSlotMenu([OPEN, separator(1), RENAME, VIEW], new Map(), WORDS)

    expect(menu.items.map((one) => [one.id, one.group])).toStrictEqual([
      [OPEN.key, '0'],
      [RENAME.key, '1'],
      [VIEW.key, '1'],
    ])
  })

  it('draws a view as a row that holds its component', () => {
    expect(createSlotMenu([VIEW], new Map(), WORDS).items).toStrictEqual([
      { id: VIEW.key, text: 'sketch', group: '0', component: VIEW.component },
    ])
  })

  it('draws a submenu as a row of its own entries, and hides a submenu with no rows', () => {
    const empty = submenu('empty', [])
    const menu = createSlotMenu([submenu('new', [OPEN, RENAME]), empty], new Map(), WORDS)

    expect(menu.items.map((one) => one.id)).toStrictEqual(['md.numen.test/submenu/new'])
    expect(menu.items[0]?.children?.map((one) => one.id)).toStrictEqual([OPEN.key, RENAME.key])
  })

  it('draws a loading row in the submenu of a command whose items are being read', () => {
    const menu = createSlotMenu([MOVE], lists([[MOVE, null]]), WORDS)

    expect(menu.items[0]?.children).toStrictEqual([
      { id: `${MOVE.key}#loading`, text: 'Loading…', isDisabled: true },
    ])
  })

  it('draws a loading row for a command no list was read for', () => {
    expect(createSlotMenu([MOVE], new Map(), WORDS).items[0]?.children?.[0]?.text).toBe('Loading…')
  })

  it('hides a command that lists no item, and leaves no group of it', () => {
    const menu = createSlotMenu([OPEN, separator(1), MOVE], lists([[MOVE, []]]), WORDS)

    expect(menu.items.map((one) => one.id)).toStrictEqual([OPEN.key])
  })

  it('draws the items of a command as its submenu, with a separator where the group changes', () => {
    const items = [
      { id: 'a', title: 'A', group: 'md.numen.ocr' },
      { id: 'b', title: 'B', group: 'md.numen.ocr' },
      { id: 'c', title: 'C', group: 'md.numen.transcription' },
    ]
    const menu = createSlotMenu([MOVE], lists([[MOVE, items]]), WORDS)

    expect(menu.items[0]?.children?.map((one) => [one.text, one.group])).toStrictEqual([
      ['A', 'md.numen.ocr'],
      ['B', 'md.numen.ocr'],
      ['C', 'md.numen.transcription'],
    ])
  })

  it('draws an item without a group without one', () => {
    const menu = createSlotMenu([MOVE], lists([[MOVE, [{ id: 'w1', title: 'Window 1' }]]]), WORDS)

    expect(menu.items[0]?.children).toStrictEqual([{ id: `${MOVE.key}#item:w1`, text: 'Window 1' }])
  })

  it('draws a shared command with one handler as one row of the title of that handler', () => {
    const menu = createSlotMenu(
      [OPEN, separator(1), CREATE_TEXT],
      lists([[CREATE_TEXT, [{ id: 'md.numen.pdf', title: 'Extract text layer' }]]]),
      WORDS,
    )

    expect(menu.items[1]).toStrictEqual({
      id: `${CREATE_TEXT.key}#item:md.numen.pdf`,
      text: 'Extract text layer',
      group: '1',
    })
    expect(menu.getPick(`${CREATE_TEXT.key}#item:md.numen.pdf`)).toStrictEqual({
      command: 'app:file.createText',
      argument: 'md.numen.pdf',
    })
  })

  it('draws a shared command with several handlers as a submenu of the handlers', () => {
    const handlers = [
      { id: 'md.numen.ocr', title: 'Recognize text' },
      { id: 'md.numen.pdf', title: 'Extract text layer' },
    ]
    const menu = createSlotMenu([CREATE_TEXT], lists([[CREATE_TEXT, handlers]]), WORDS)

    expect(menu.items[0]?.text).toBe('Create text')
    expect(menu.items[0]?.children?.map((one) => one.text)).toStrictEqual([
      'Recognize text',
      'Extract text layer',
    ])
  })

  it('draws the one item of a handler of a shared command in a submenu', () => {
    const items = [{ id: 'md.numen.ocr:v1', title: 'Recognition', group: 'md.numen.ocr' }]
    const menu = createSlotMenu([CREATE_TEXT], lists([[CREATE_TEXT, items]]), WORDS)

    expect(menu.items[0]?.children?.map((one) => one.text)).toStrictEqual(['Recognition'])
  })

  it('hides a shared command that lists no item, and leaves no group of it', () => {
    const menu = createSlotMenu(
      [OPEN, separator(1), CREATE_TEXT],
      lists([[CREATE_TEXT, []]]),
      WORDS,
    )

    expect(menu.items.map((one) => one.id)).toStrictEqual([OPEN.key])
  })

  it('draws nothing for a slot without entries', () => {
    expect(createSlotMenu([], new Map(), WORDS).items).toStrictEqual([])
  })
})

describe('the pick of a row', () => {
  const entries = [OPEN, separator(1), submenu('new', [RENAME]), MOVE, VIEW]
  const menu = createSlotMenu(entries, lists([[MOVE, [{ id: 'w1', title: 'Window 1' }]]]), WORDS)

  it('runs the command of a row, inside a submenu too', () => {
    expect(menu.getPick(OPEN.key)).toStrictEqual({ command: OPEN.command })
    expect(menu.getPick(RENAME.key)).toStrictEqual({ command: RENAME.command })
  })

  it('runs the command of an item with the item as its argument', () => {
    expect(menu.getPick(`${MOVE.key}#item:w1`)).toStrictEqual({
      command: MOVE.command,
      argument: 'w1',
    })
  })

  it('runs nothing for the row of a command with items, a submenu, a view or an unknown id', () => {
    for (const id of [MOVE.key, 'md.numen.test/submenu/new', VIEW.key, 'separator/1', 'none']) {
      expect(menu.getPick(id)).toBeUndefined()
    }
  })

  it('runs nothing for the loading and the empty row', () => {
    const loading = createSlotMenu([MOVE], new Map(), WORDS)

    expect(loading.getPick(`${MOVE.key}#loading`)).toBeUndefined()
  })
})

describe('the icon of a row', () => {
  it('is the icon of its entry, and of the entry of a shared command drawn as one row', () => {
    const shared = { ...CREATE_TEXT, icon: 'text-icon' }
    const menu = createSlotMenu(
      [OPEN, RENAME, shared],
      lists([[shared, [{ id: 'md.numen.pdf', title: 'Extract text layer' }]]]),
      WORDS,
    )

    expect(menu.getIcon(OPEN.key)).toBe('open-icon')
    expect(menu.getIcon(RENAME.key)).toBeUndefined()
    expect(menu.getIcon(`${shared.key}#item:md.numen.pdf`)).toBe('text-icon')
  })
})

const createHandlers = () => ({
  onChange: vi.fn(),
  onShow: vi.fn(),
  onCancel: vi.fn(),
  onError: vi.fn(),
  events: new EventTarget(),
})

const createKeyEvent = (key: string): Event => Object.assign(new Event('keydown'), { key })

describe('readCommandItemLists', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it('reads the items of every command with items, inside submenus too', async () => {
    const listItems = vi.fn(async (id: string) => [{ id: `${id}-1`, title: 'One' }])
    const handlers = createHandlers()

    readCommandItemLists([OPEN, submenu('new', [MOVE])], listItems, handlers)
    await vi.waitFor(() => expect(handlers.onChange).toHaveBeenCalledTimes(2))

    expect(listItems.mock.calls.map(([id]) => id)).toStrictEqual([MOVE.command])
    expect(handlers.onChange.mock.calls[0]?.[0]).toStrictEqual(new Map([[MOVE.key, null]]))
    expect(handlers.onChange.mock.calls[1]?.[0]).toStrictEqual(
      new Map([[MOVE.key, [{ id: `${MOVE.command}-1`, title: 'One' }]]]),
    )
  })

  it('shows the menu at once when no command of it lists items', () => {
    const handlers = createHandlers()

    readCommandItemLists([OPEN, RENAME], vi.fn(), handlers)

    expect(handlers.onShow).toHaveBeenCalledTimes(1)
  })

  it('shows the menu once every list is read, before the hold ends', async () => {
    vi.useFakeTimers()
    const resolvers: ((items: readonly CommandItem[]) => void)[] = []
    const handlers = createHandlers()
    readCommandItemLists(
      [MOVE, CREATE_TEXT],
      () => new Promise((resolve) => resolvers.push(resolve)),
      handlers,
    )

    resolvers[0]?.([{ id: 'w1', title: 'Window 1' }])
    await vi.advanceTimersByTimeAsync(0)
    expect(handlers.onShow).not.toHaveBeenCalled()
    resolvers[1]?.([])
    await vi.advanceTimersByTimeAsync(0)

    expect(handlers.onShow).toHaveBeenCalledTimes(1)
    expect(handlers.onChange).toHaveBeenLastCalledWith(
      new Map([
        [MOVE.key, [{ id: 'w1', title: 'Window 1' }]],
        [CREATE_TEXT.key, []],
      ]),
    )
    await vi.advanceTimersByTimeAsync(MENU_HOLD_MS)
    expect(handlers.onShow).toHaveBeenCalledTimes(1)
  })

  it('shows the menu with its loading rows when the lists are not read within the hold', async () => {
    vi.useFakeTimers()
    const handlers = createHandlers()
    readCommandItemLists([MOVE], () => new Promise(() => undefined), handlers)

    await vi.advanceTimersByTimeAsync(MENU_HOLD_MS - 1)
    expect(handlers.onShow).not.toHaveBeenCalled()
    await vi.advanceTimersByTimeAsync(1)

    expect(handlers.onShow).toHaveBeenCalledTimes(1)
    expect(handlers.onChange).toHaveBeenLastCalledWith(new Map([[MOVE.key, null]]))
  })

  it.each([
    ['Escape', createKeyEvent('Escape')],
    ['a pointer press', new Event('pointerdown')],
  ])('cancels the open on %s before the menu is shown', async (_name, event) => {
    vi.useFakeTimers()
    let resolve: (items: readonly CommandItem[]) => void = () => undefined
    const handlers = createHandlers()
    readCommandItemLists([MOVE], () => new Promise((done) => (resolve = done)), handlers)

    handlers.events.dispatchEvent(event)
    resolve([{ id: 'w1', title: 'Window 1' }])
    await vi.advanceTimersByTimeAsync(MENU_HOLD_MS)

    expect(handlers.onCancel).toHaveBeenCalledTimes(1)
    expect(handlers.onShow).not.toHaveBeenCalled()
    expect(handlers.onChange).toHaveBeenCalledTimes(1)
  })

  it('keeps an open menu on another key and after it is shown', async () => {
    vi.useFakeTimers()
    const handlers = createHandlers()
    readCommandItemLists([MOVE], () => new Promise(() => undefined), handlers)

    handlers.events.dispatchEvent(createKeyEvent('ArrowDown'))
    await vi.advanceTimersByTimeAsync(MENU_HOLD_MS)
    handlers.events.dispatchEvent(createKeyEvent('Escape'))
    handlers.events.dispatchEvent(new Event('pointerdown'))

    expect(handlers.onShow).toHaveBeenCalledTimes(1)
    expect(handlers.onCancel).not.toHaveBeenCalled()
  })

  it('stops reporting once disposed', async () => {
    vi.useFakeTimers()
    let resolve: (items: readonly CommandItem[]) => void = () => undefined
    const handlers = createHandlers()
    const itemSubscription = readCommandItemLists(
      [MOVE],
      () => new Promise((done) => (resolve = done)),
      handlers,
    )

    itemSubscription.dispose()
    resolve([{ id: 'w1', title: 'Window 1' }])
    await vi.advanceTimersByTimeAsync(MENU_HOLD_MS)
    handlers.events.dispatchEvent(createKeyEvent('Escape'))

    expect(handlers.onChange).toHaveBeenCalledTimes(1)
    expect(handlers.onShow).not.toHaveBeenCalled()
    expect(handlers.onCancel).not.toHaveBeenCalled()
  })

  it('reports a failed read and lists no item for its command', async () => {
    const error = new Error('core is down')
    const handlers = createHandlers()

    readCommandItemLists([MOVE], () => Promise.reject(error), handlers)
    await vi.waitFor(() => expect(handlers.onError).toHaveBeenCalledWith(error))

    expect(handlers.onChange).toHaveBeenLastCalledWith(new Map([[MOVE.key, []]]))
    expect(handlers.onShow).toHaveBeenCalledTimes(1)
  })
})
