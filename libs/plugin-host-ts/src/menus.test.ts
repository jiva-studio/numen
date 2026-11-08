import { describe, expect, it, vi } from 'vitest'
import {
  compareEntries,
  createMenus,
  type CommandLook,
  type MenuPlacement,
  type PlacedCommand,
} from './menus'

const ATLAS = 'org.example.atlas'
const BOOKS = 'org.example.books'

const look = (title: string, icon = 'star'): CommandLook => ({ title, icon })

function setup(
  placements: readonly MenuPlacement[] | undefined,
  commands: Record<string, CommandLook> = {},
) {
  const state = { placements, commands }
  const reportIssue = vi.fn<(pluginId: string, message: string) => void>()
  const menus = createMenus({
    getPlacements: () => state.placements,
    findCommand: (fullId) => state.commands[fullId],
    reportIssue,
  })
  return { menus, state, reportIssue }
}

const ids = (entries: readonly PlacedCommand[]) => entries.map((one) => one.command)

function listPermutations<T>(items: readonly T[]): T[][] {
  if (items.length <= 1) return [[...items]]
  return items.flatMap((item, index) =>
    listPermutations([...items.slice(0, index), ...items.slice(index + 1)]).map((rest) => [
      item,
      ...rest,
    ]),
  )
}

describe('the entries of a place', () => {
  it('has none where no plugin places a command in it', () => {
    const { menus } = setup([{ pluginId: ATLAS, location: 'titleBar', command: 'atlas.open' }], {
      [`${ATLAS}:atlas.open`]: look('Open atlas'),
    })

    expect(menus.getEntries('searchResult')).toEqual([])
  })

  it('lists the groups alphabetically and the entries with no group after them', () => {
    const { menus } = setup(
      [
        { pluginId: ATLAS, location: 'titleBar', command: 'atlas.plain' },
        { pluginId: ATLAS, location: 'titleBar', command: 'atlas.fresh', group: 'new' },
        { pluginId: ATLAS, location: 'titleBar', command: 'atlas.goto', group: 'navigate' },
      ],
      {
        [`${ATLAS}:atlas.plain`]: look('Plain'),
        [`${ATLAS}:atlas.fresh`]: look('Fresh'),
        [`${ATLAS}:atlas.goto`]: look('Go to'),
      },
    )

    expect(ids(menus.getEntries('titleBar'))).toEqual([
      `${ATLAS}:atlas.goto`,
      `${ATLAS}:atlas.fresh`,
      `${ATLAS}:atlas.plain`,
    ])
  })

  it('sorts group names as plain text, capitals before lowercase letters', () => {
    const { menus } = setup(
      [
        { pluginId: ATLAS, location: 'titleBar', command: 'atlas.low', group: 'alpha' },
        { pluginId: ATLAS, location: 'titleBar', command: 'atlas.high', group: 'Zeta' },
      ],
      { [`${ATLAS}:atlas.low`]: look('Low'), [`${ATLAS}:atlas.high`]: look('High') },
    )

    expect(ids(menus.getEntries('titleBar'))).toEqual([`${ATLAS}:atlas.high`, `${ATLAS}:atlas.low`])
  })

  it('sorts the entries of a group by order, lowest first, and those with no order last', () => {
    const { menus } = setup(
      [
        { pluginId: ATLAS, location: 'titleBar', command: 'atlas.none', group: 'g' },
        { pluginId: ATLAS, location: 'titleBar', command: 'atlas.twenty', group: 'g', order: 20 },
        { pluginId: ATLAS, location: 'titleBar', command: 'atlas.ten', group: 'g', order: 10 },
        { pluginId: ATLAS, location: 'titleBar', command: 'atlas.zero', group: 'g', order: 0 },
      ],
      {
        [`${ATLAS}:atlas.none`]: look('None'),
        [`${ATLAS}:atlas.twenty`]: look('Twenty'),
        [`${ATLAS}:atlas.ten`]: look('Ten'),
        [`${ATLAS}:atlas.zero`]: look('Zero'),
      },
    )

    expect(ids(menus.getEntries('titleBar'))).toEqual([
      `${ATLAS}:atlas.zero`,
      `${ATLAS}:atlas.ten`,
      `${ATLAS}:atlas.twenty`,
      `${ATLAS}:atlas.none`,
    ])
  })

  it('sorts the entries with no group by order and then by command id', () => {
    const { menus } = setup(
      [
        { pluginId: ATLAS, location: 'titleBar', command: 'atlas.b' },
        { pluginId: ATLAS, location: 'titleBar', command: 'atlas.c', order: 5 },
        { pluginId: ATLAS, location: 'titleBar', command: 'atlas.a' },
      ],
      {
        [`${ATLAS}:atlas.a`]: look('A'),
        [`${ATLAS}:atlas.b`]: look('B'),
        [`${ATLAS}:atlas.c`]: look('C'),
      },
    )

    expect(ids(menus.getEntries('titleBar'))).toEqual([
      `${ATLAS}:atlas.c`,
      `${ATLAS}:atlas.a`,
      `${ATLAS}:atlas.b`,
    ])
  })

  it('breaks a tie by the full command id as plain text, whatever the locale', () => {
    const { menus } = setup(
      [
        { pluginId: BOOKS, location: 'titleBar', command: 'x.y', group: 'g' },
        { pluginId: ATLAS, location: 'titleBar', command: 'x.y', group: 'g' },
        { pluginId: ATLAS, location: 'titleBar', command: 'app:a.x', group: 'h' },
        { pluginId: ATLAS, location: 'titleBar', command: 'app:B.x', group: 'h' },
      ],
      {
        [`${ATLAS}:x.y`]: look('Atlas'),
        [`${BOOKS}:x.y`]: look('Books'),
        'app:a.x': look('Lower'),
        'app:B.x': look('Upper'),
      },
    )

    expect(ids(menus.getEntries('titleBar'))).toEqual([
      `${ATLAS}:x.y`,
      `${BOOKS}:x.y`,
      'app:B.x',
      'app:a.x',
    ])
  })

  it('gives the same result for the placements in every order of listing', () => {
    const placements: MenuPlacement[] = [
      { pluginId: ATLAS, location: 'titleBar', command: 'atlas.one', group: 'b', order: 2 },
      { pluginId: BOOKS, location: 'titleBar', command: 'books.two', group: 'b', order: 2 },
      { pluginId: ATLAS, location: 'titleBar', command: 'atlas.three', group: 'a' },
      { pluginId: BOOKS, location: 'titleBar', command: 'app:note.create', order: 1 },
      { pluginId: ATLAS, location: 'titleBar', command: 'atlas.five' },
    ]
    const commands = {
      [`${ATLAS}:atlas.one`]: look('One'),
      [`${BOOKS}:books.two`]: look('Two'),
      [`${ATLAS}:atlas.three`]: look('Three'),
      'app:note.create': look('Create note'),
      [`${ATLAS}:atlas.five`]: look('Five'),
    }
    const expected = ids(setup(placements, commands).menus.getEntries('titleBar'))
    expect(expected).toHaveLength(5)

    for (const permutation of listPermutations(placements)) {
      expect(ids(setup(permutation, commands).menus.getEntries('titleBar'))).toEqual(expected)
    }
  })

  it('has no entry of a plugin whose placements are no longer listed', () => {
    const { menus, state } = setup(
      [
        { pluginId: ATLAS, location: 'titleBar', command: 'atlas.open' },
        { pluginId: ATLAS, location: 'searchResult', command: 'atlas.open' },
        { pluginId: BOOKS, location: 'titleBar', command: 'books.open' },
      ],
      { [`${ATLAS}:atlas.open`]: look('Open atlas'), [`${BOOKS}:books.open`]: look('Open books') },
    )

    state.placements = state.placements!.filter((one) => one.pluginId !== ATLAS)

    expect(ids(menus.getEntries('titleBar'))).toEqual([`${BOOKS}:books.open`])
    expect(menus.getEntries('searchResult')).toEqual([])
  })

  it('returns an enabled plugin to the same positions', () => {
    const all: MenuPlacement[] = [
      { pluginId: ATLAS, location: 'titleBar', command: 'atlas.open', group: 'g', order: 2 },
      { pluginId: BOOKS, location: 'titleBar', command: 'books.open', group: 'g', order: 1 },
      { pluginId: BOOKS, location: 'titleBar', command: 'books.last', group: 'g' },
    ]
    const { menus, state } = setup(all, {
      [`${ATLAS}:atlas.open`]: look('Open atlas'),
      [`${BOOKS}:books.open`]: look('Open books'),
      [`${BOOKS}:books.last`]: look('Last'),
    })
    const before = menus.getEntries('titleBar')

    state.placements = all.filter((one) => one.pluginId !== ATLAS)
    expect(menus.getEntries('titleBar')).toHaveLength(2)
    state.placements = [...all].reverse()

    expect(menus.getEntries('titleBar')).toEqual(before)
  })

  it('leaves out a command the window does not list, and reports it once', () => {
    const { menus, reportIssue } = setup(
      [
        { pluginId: ATLAS, location: 'titleBar', command: 'atlas.ghost' },
        { pluginId: ATLAS, location: 'titleBar', command: 'atlas.open' },
      ],
      { [`${ATLAS}:atlas.open`]: look('Open atlas') },
    )

    const first = menus.getEntries('titleBar')
    const second = menus.getEntries('titleBar')

    expect(ids(first)).toEqual([`${ATLAS}:atlas.open`])
    expect(ids(second)).toEqual([`${ATLAS}:atlas.open`])
    expect(reportIssue).toHaveBeenCalledTimes(1)
    expect(reportIssue.mock.calls[0]?.[0]).toBe(ATLAS)
  })

  it('leaves out an unknown place and reports it once', () => {
    const { menus, reportIssue } = setup(
      [{ pluginId: ATLAS, location: 'sidebar', command: 'atlas.open' }],
      { [`${ATLAS}:atlas.open`]: look('Open atlas') },
    )

    expect(menus.getEntries('titleBar')).toEqual([])
    expect(menus.getEntries('searchResult')).toEqual([])
    expect(menus.getEntries('titleBar')).toEqual([])
    expect(reportIssue).toHaveBeenCalledTimes(1)
    expect(reportIssue.mock.calls[0]?.[0]).toBe(ATLAS)
  })

  it('has none for a place no window draws', () => {
    const { menus } = setup([{ pluginId: ATLAS, location: 'sidebar', command: 'atlas.open' }], {
      [`${ATLAS}:atlas.open`]: look('Open atlas'),
    })

    expect(menus.getEntries('sidebar')).toEqual([])
  })

  it('gives one command in two places the group and order of each placement', () => {
    const { menus } = setup(
      [
        {
          pluginId: ATLAS,
          location: 'titleBar',
          command: 'atlas.open',
          group: 'navigate',
          order: 10,
        },
        {
          pluginId: ATLAS,
          location: 'searchResult',
          command: 'atlas.open',
          group: 'show',
          order: 5,
        },
      ],
      { [`${ATLAS}:atlas.open`]: look('Open atlas') },
    )

    const [inBar] = menus.getEntries('titleBar')
    const [inResult] = menus.getEntries('searchResult')

    expect([inBar?.group, inBar?.order]).toEqual(['navigate', 10])
    expect([inResult?.group, inResult?.order]).toEqual(['show', 5])
  })

  it('carries the title, icon and shortcut of the command and files a short id under its plugin', () => {
    const { menus } = setup(
      [
        { pluginId: ATLAS, location: 'titleBar', command: 'atlas.open' },
        { pluginId: ATLAS, location: 'titleBar', command: 'app:note.create', order: 1 },
      ],
      {
        [`${ATLAS}:atlas.open`]: { title: 'Open atlas', icon: 'map', shortcut: 'mod+shift+m' },
        'app:note.create': { title: 'New note', icon: 'file-plus' },
      },
    )

    expect(menus.getEntries('titleBar')).toEqual([
      {
        pluginId: ATLAS,
        command: 'app:note.create',
        group: '',
        order: 1,
        title: 'New note',
        icon: 'file-plus',
      },
      {
        pluginId: ATLAS,
        command: `${ATLAS}:atlas.open`,
        group: '',
        title: 'Open atlas',
        icon: 'map',
        shortcut: 'mod+shift+m',
      },
    ])
  })

  it('leaves a command with no icon out of the title bar and keeps it in a search result', () => {
    const { menus, reportIssue } = setup(
      [
        { pluginId: ATLAS, location: 'titleBar', command: 'atlas.open' },
        { pluginId: ATLAS, location: 'searchResult', command: 'atlas.open' },
      ],
      { [`${ATLAS}:atlas.open`]: { title: 'Open atlas' } },
    )

    expect(menus.getEntries('titleBar')).toEqual([])
    expect(ids(menus.getEntries('searchResult'))).toEqual([`${ATLAS}:atlas.open`])
    menus.getEntries('titleBar')
    expect(reportIssue).toHaveBeenCalledTimes(1)
  })

  it('shows a command the window lists before its plugin has registered a handler', () => {
    const { menus, reportIssue } = setup(
      [{ pluginId: ATLAS, location: 'searchResult', command: 'atlas.open' }],
      { [`${ATLAS}:atlas.open`]: look('Open atlas') },
    )

    expect(ids(menus.getEntries('searchResult'))).toEqual([`${ATLAS}:atlas.open`])
    expect(reportIssue).not.toHaveBeenCalled()
  })

  it('has no entries and reports nothing before the first list has loaded', () => {
    const { menus, state, reportIssue } = setup(undefined, {
      [`${ATLAS}:atlas.open`]: look('Open atlas'),
    })

    expect(menus.getEntries('titleBar')).toEqual([])
    expect(menus.getEntries('searchResult')).toEqual([])
    expect(reportIssue).not.toHaveBeenCalled()

    state.placements = [{ pluginId: ATLAS, location: 'titleBar', command: 'atlas.open' }]
    expect(ids(menus.getEntries('titleBar'))).toEqual([`${ATLAS}:atlas.open`])
  })

  it('leaves a command that needs a note out of the title bar and reports it', () => {
    const { menus, reportIssue } = setup(
      [
        { pluginId: ATLAS, location: 'titleBar', command: 'atlas.show' },
        { pluginId: ATLAS, location: 'searchResult', command: 'atlas.show' },
      ],
      { [`${ATLAS}:atlas.show`]: { title: 'Show note', icon: 'map', scope: 'note' } },
    )

    expect(menus.getEntries('titleBar')).toEqual([])
    expect(ids(menus.getEntries('searchResult'))).toEqual([`${ATLAS}:atlas.show`])
    menus.getEntries('titleBar')
    expect(reportIssue).toHaveBeenCalledTimes(1)
  })

  it('returns a command placed in one place by two plugins once and reports the other', () => {
    const { menus, reportIssue } = setup(
      [
        { pluginId: BOOKS, location: 'titleBar', command: 'app:note.create', order: 2 },
        { pluginId: ATLAS, location: 'titleBar', command: 'app:note.create', order: 1 },
      ],
      { 'app:note.create': look('New note', 'file-plus') },
    )

    const entries = menus.getEntries('titleBar')
    menus.getEntries('titleBar')

    expect(entries).toHaveLength(1)
    expect(entries[0]?.pluginId).toBe(ATLAS)
    expect(reportIssue).toHaveBeenCalledTimes(1)
    expect(reportIssue.mock.calls[0]?.[0]).toBe(BOOKS)
  })

  it('returns an entry listed twice for one place once', () => {
    const placement: MenuPlacement = {
      pluginId: ATLAS,
      location: 'titleBar',
      command: 'atlas.open',
    }
    const { menus } = setup([placement, { ...placement }], {
      [`${ATLAS}:atlas.open`]: look('Open atlas'),
    })

    expect(ids(menus.getEntries('titleBar'))).toEqual([`${ATLAS}:atlas.open`])
  })

  it('reads the placements and the commands again on every call', () => {
    const { menus, state } = setup([
      { pluginId: ATLAS, location: 'titleBar', command: 'atlas.open' },
    ])
    expect(menus.getEntries('titleBar')).toEqual([])

    state.commands = { [`${ATLAS}:atlas.open`]: look('Open atlas') }

    expect(ids(menus.getEntries('titleBar'))).toEqual([`${ATLAS}:atlas.open`])
  })
})

describe('the comparison of two entries', () => {
  const entry = (command: string, group = '', order?: number): PlacedCommand => ({
    pluginId: ATLAS,
    command,
    group,
    order,
    title: command,
  })

  it('puts an entry with a group before one with none', () => {
    expect(compareEntries(entry('x:a', 'g'), entry('x:b'))).toBeLessThan(0)
    expect(compareEntries(entry('x:b'), entry('x:a', 'g'))).toBeGreaterThan(0)
  })

  it('compares code units, not the words of a language', () => {
    expect(compareEntries(entry('app:B.x'), entry('app:a.x'))).toBeLessThan(0)
    expect(compareEntries(entry('x:a', 'B'), entry('x:b', 'a'))).toBeLessThan(0)
  })

  it('puts an entry with an order before one with none and is zero for the same entry', () => {
    expect(compareEntries(entry('x:b', 'g', 9), entry('x:a', 'g'))).toBeLessThan(0)
    expect(compareEntries(entry('x:a', 'g', 1), entry('x:a', 'g', 1))).toBe(0)
  })
})
