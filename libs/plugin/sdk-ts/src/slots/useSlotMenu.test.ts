/**
 * The open of a slot menu: the entries read, the item lists read, the menu shown or cancelled, a row
 * picked. Every command and target is invented.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { CommandItem } from '../commands'
import type { SlotCommand, SlotItem } from './slots'
import { MENU_HOLD_MS, useSlotMenu, type SlotMenuRequest } from './useSlotMenu'

const command = (id: string, extra: Partial<SlotCommand> = {}): SlotCommand => ({
  kind: 'command',
  key: `org.example.test/command/${id}`,
  pluginId: 'org.example.test',
  command: `org.example.test:${id}`,
  title: id,
  shortcut: '',
  hasItems: false,
  ...extra,
})

const OPEN = command('open')
const RENAME = command('rename')
const MOVE = command('moveToWindow', { hasItems: true })
const SPLIT = command('split', { hasItems: true })
const TARGET = 'notes/plan.md'
const WINDOW_ONE: CommandItem = { id: 'w1', title: 'Window 1' }

const createKeyEvent = (key: string): Event => Object.assign(new Event('keydown'), { key })

const setUp = (listItems: (command: string, target: string) => Promise<readonly CommandItem[]>) => {
  const events = new EventTarget()
  const onError = vi.fn()
  const spy = vi.fn(listItems)
  const state = useSlotMenu<string>({
    listItems: spy,
    words: { loadingText: 'Loading…' },
    onError,
    events,
  })
  const open = (entries: readonly SlotItem[], target = TARGET) =>
    state.open(() => ({ target, entries }))
  return { state, events, onError, listItems: spy, open }
}

const getRowIds = (state: ReturnType<typeof setUp>['state']) =>
  state.menu.value?.items.map((one) => one.id) ?? null

describe('useSlotMenu', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it('shows a menu with no command that lists items once its entries are read', async () => {
    const { state, open, listItems } = setUp(vi.fn())

    open([OPEN, RENAME])
    await vi.waitFor(() => expect(state.target.value).toBe(TARGET))

    expect(getRowIds(state)).toStrictEqual([OPEN.key, RENAME.key])
    expect(listItems).not.toHaveBeenCalled()
  })

  it('reads the items of every command over the target, and shows the menu once all are read', async () => {
    vi.useFakeTimers()
    const resolvers: ((items: readonly CommandItem[]) => void)[] = []
    const { state, open, listItems } = setUp(
      () => new Promise((resolve) => resolvers.push(resolve)),
    )

    open([MOVE, SPLIT])
    await vi.advanceTimersByTimeAsync(0)
    resolvers[0]?.([WINDOW_ONE])
    await vi.advanceTimersByTimeAsync(0)
    expect(state.menu.value).toBeNull()
    resolvers[1]?.([{ id: 'left', title: 'Left' }])
    await vi.advanceTimersByTimeAsync(0)

    expect(listItems.mock.calls).toStrictEqual([
      [MOVE.command, TARGET],
      [SPLIT.command, TARGET],
    ])
    expect(state.menu.value?.items[0]?.children?.map((one) => one.text)).toStrictEqual(['Window 1'])
  })

  it('shows the menu with loading rows 150 ms after the open', async () => {
    vi.useFakeTimers()
    const { state, open } = setUp(() => new Promise(() => undefined))

    open([MOVE])
    await vi.advanceTimersByTimeAsync(149)
    expect(state.menu.value).toBeNull()
    await vi.advanceTimersByTimeAsync(1)

    expect(MENU_HOLD_MS).toBe(150)
    expect(state.menu.value?.items[0]?.children?.[0]?.text).toBe('Loading…')
  })

  it('counts the hold from the open, while the entries are still read', async () => {
    vi.useFakeTimers()
    let answer: (request: SlotMenuRequest<string>) => void = () => undefined
    const { state } = setUp(() => new Promise(() => undefined))

    state.open(() => new Promise((resolve) => (answer = resolve)))
    await vi.advanceTimersByTimeAsync(150)
    expect(state.menu.value).toBeNull()
    answer({ target: TARGET, entries: [MOVE] })
    await vi.advanceTimersByTimeAsync(0)

    expect(state.menu.value?.items[0]?.children?.[0]?.text).toBe('Loading…')
  })

  it.each([
    ['Escape', createKeyEvent('Escape')],
    ['a pointer press', new Event('pointerdown')],
  ])('cancels the open on %s while the lists are read', async (_name, event) => {
    vi.useFakeTimers()
    let resolve: (items: readonly CommandItem[]) => void = () => undefined
    const { state, open, events } = setUp(() => new Promise((done) => (resolve = done)))

    open([MOVE])
    await vi.advanceTimersByTimeAsync(0)
    events.dispatchEvent(event)
    resolve([WINDOW_ONE])
    await vi.advanceTimersByTimeAsync(150)

    expect(state.menu.value).toBeNull()
  })

  it('cancels the open on Escape while its entries are read', async () => {
    vi.useFakeTimers()
    let answer: (request: SlotMenuRequest<string>) => void = () => undefined
    const { state, events, listItems } = setUp(async () => [WINDOW_ONE])

    state.open(() => new Promise((resolve) => (answer = resolve)))
    events.dispatchEvent(createKeyEvent('Escape'))
    answer({ target: TARGET, entries: [MOVE] })
    await vi.advanceTimersByTimeAsync(150)

    expect(state.menu.value).toBeNull()
    expect(listItems).not.toHaveBeenCalled()
  })

  it('keeps an open menu on another key and after it is shown', async () => {
    vi.useFakeTimers()
    const { state, open, events } = setUp(() => new Promise(() => undefined))

    open([MOVE])
    events.dispatchEvent(createKeyEvent('ArrowDown'))
    await vi.advanceTimersByTimeAsync(150)
    events.dispatchEvent(createKeyEvent('Escape'))
    events.dispatchEvent(new Event('pointerdown'))

    expect(state.target.value).toBe(TARGET)
  })

  it('shows no menu whose rows all list nothing, so the next open opens again', async () => {
    const { state, open } = setUp(async () => [])

    open([MOVE])
    await vi.waitFor(() => expect(state.menu.value).toBeNull())
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(state.target.value).toBeNull()

    open([OPEN])
    await vi.waitFor(() => expect(getRowIds(state)).toStrictEqual([OPEN.key]))
  })

  it('opens nothing for no entries', async () => {
    const { state, open } = setUp(vi.fn())

    open([])
    await new Promise((resolve) => setTimeout(resolve, 0))

    expect(state.menu.value).toBeNull()
  })

  it('reports a failed read and lists no item for its command', async () => {
    const error = new Error('core is down')
    const { state, open, onError } = setUp(() => Promise.reject(error))

    open([OPEN, MOVE])
    await vi.waitFor(() => expect(onError).toHaveBeenCalledWith(error))

    await vi.waitFor(() => expect(getRowIds(state)).toStrictEqual([OPEN.key]))
  })

  it('reports a failed read of the entries and opens nothing', async () => {
    const error = new Error('no note kind')
    const { state, onError } = setUp(vi.fn())

    state.open(() => Promise.reject(error))
    await vi.waitFor(() => expect(onError).toHaveBeenCalledWith(error))

    expect(state.menu.value).toBeNull()
  })

  it('drops the lists of an open replaced by a later one', async () => {
    vi.useFakeTimers()
    const resolvers: ((items: readonly CommandItem[]) => void)[] = []
    const { state, open } = setUp(() => new Promise((resolve) => resolvers.push(resolve)))

    open([MOVE], 'notes/first.md')
    await vi.advanceTimersByTimeAsync(0)
    open([MOVE], 'notes/second.md')
    await vi.advanceTimersByTimeAsync(0)
    resolvers[0]?.([WINDOW_ONE])
    await vi.advanceTimersByTimeAsync(0)
    expect(state.menu.value).toBeNull()
    resolvers[1]?.([{ id: 'w2', title: 'Window 2' }])
    await vi.advanceTimersByTimeAsync(0)

    expect(state.target.value).toBe('notes/second.md')
    expect(state.menu.value?.items[0]?.children?.map((one) => one.text)).toStrictEqual(['Window 2'])
  })

  it('returns the command and item of a picked row with the target, and closes the menu', async () => {
    const { state, open } = setUp(async () => [WINDOW_ONE])
    open([OPEN, MOVE])
    await vi.waitFor(() => expect(state.menu.value).not.toBeNull())

    const choice = state.pick(`${MOVE.key}#item:w1`)

    expect(choice).toStrictEqual({ command: MOVE.command, itemId: 'w1', target: TARGET })
    expect(state.menu.value).toBeNull()
  })

  it('returns nothing for a row that runs nothing, and closes the menu', async () => {
    const { state, open } = setUp(vi.fn())
    open([OPEN])
    await vi.waitFor(() => expect(state.menu.value).not.toBeNull())

    expect(state.pick('no-such-row')).toBeUndefined()
    expect(state.menu.value).toBeNull()
  })
})
