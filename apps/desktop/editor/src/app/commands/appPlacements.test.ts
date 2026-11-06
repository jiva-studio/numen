/** The places the application puts its own commands in: one list, merged with the plugins'. */
import { describe, expect, it } from 'vitest'
import type { MenuPlacement } from '@numen/plugin-host'
import { createCommands } from '@/features/command-palette'
import { COMMAND } from '@/shared/commandIds'
import { WORDS } from '@/shared/words'
import { APP_PLACEMENTS } from './appPlacements'
import { createPluginMenus } from './pluginMenus'

const TIDE = 'org.example.tide'
const REEF = 'org.example.reef'

const tidePlacement: MenuPlacement = {
  pluginId: TIDE,
  location: 'titleBar.commands',
  command: 'tide.open',
  group: 'navigate',
}
const reefPlacement: MenuPlacement = {
  pluginId: REEF,
  location: 'titleBar.commands',
  command: 'reef.open',
  group: 'navigate',
}

function setup(placements: readonly MenuPlacement[], getAppPlacements?: () => MenuPlacement[]) {
  return createPluginMenus({
    getPlacements: () => placements,
    getPluginCommands: () => [
      { id: `${TIDE}:tide.open`, title: 'Open tide', icon: 'waves' },
      { id: `${REEF}:reef.open`, title: 'Open reef', icon: 'map' },
      { id: `${TIDE}:tide.alpha`, title: 'Open alpha', icon: 'waves' },
      { id: `${TIDE}:tide.zeta`, title: 'Open zeta', icon: 'waves' },
    ],
    getDeclaredCommands: () => [],
    getEditorCommands: () => createCommands(WORDS),
    ...(getAppPlacements ? { getAppPlacements } : {}),
  })
}

const getCommands = (menus: ReturnType<typeof setup>) =>
  menus.getEntries('titleBar.commands').map((one) => one.command)

describe('the placements of the application', () => {
  it('are one list that places the settings command in the title bar in a group of its own', () => {
    expect(APP_PLACEMENTS).toEqual([
      {
        pluginId: 'app',
        location: 'titleBar.commands',
        command: COMMAND.openSettings,
        group: 'settings',
      },
    ])
  })

  it('name no plugin and keep apart from the placements the plugins give', () => {
    const plugins = [tidePlacement]

    setup(plugins)

    expect(plugins).toEqual([tidePlacement])
    for (const one of APP_PLACEMENTS) expect(one.pluginId).not.toMatch(/^md\.numen\./)
  })

  it('give the settings entry with the gear icon, the title and no plugin of its own', () => {
    const menus = setup([])

    expect(menus.getEntries('titleBar.commands')).toMatchObject([
      { command: 'app:settings.open', title: 'Open settings', icon: 'settings', group: 'settings' },
    ])
  })
})

describe('the settings entry among the entries of the plugins', () => {
  it('stays last when a plugin places a command in a group that sorts before settings', () => {
    const menus = setup([
      tidePlacement,
      { pluginId: TIDE, location: 'titleBar.commands', command: 'tide.alpha', group: 'alpha' },
      { pluginId: TIDE, location: 'titleBar.commands', command: 'tide.zeta', group: 'sa' },
    ])

    expect(getCommands(menus).at(-1)).toBe('app:settings.open')
    expect(getCommands(menus)).toHaveLength(4)
  })

  it('sorts the groups by name before the order inside them and keeps settings last', () => {
    const menus = setup([
      { ...tidePlacement, order: 99 },
      {
        pluginId: REEF,
        location: 'titleBar.commands',
        command: 'reef.open',
        group: 'new',
        order: 1,
      },
    ])

    expect(getCommands(menus)).toEqual([
      `${TIDE}:tide.open`,
      `${REEF}:reef.open`,
      'app:settings.open',
    ])
  })

  it('gives the same entries whatever order the plugins load in', () => {
    const placements = [
      tidePlacement,
      reefPlacement,
      { pluginId: TIDE, location: 'titleBar.commands', command: 'tide.alpha', group: 'alpha' },
    ]
    const expected = getCommands(setup(placements))
    const orders = [
      [placements[2]!, placements[1]!, placements[0]!],
      [placements[1]!, placements[0]!, placements[2]!],
      [placements[1]!, placements[2]!, placements[0]!],
    ]

    expect(expected.at(-1)).toBe('app:settings.open')
    for (const order of orders) expect(getCommands(setup(order))).toEqual(expected)
  })

  it('merges the list of the application in with a plugin placement and one resolution', () => {
    const menus = setup([tidePlacement], () => [
      {
        pluginId: 'app',
        location: 'titleBar.commands',
        command: COMMAND.openSettings,
        group: 'settings',
      },
    ])

    expect(getCommands(menus)).toEqual([`${TIDE}:tide.open`, 'app:settings.open'])
  })

  it('places nothing of the application in the search result place', () => {
    expect(setup([]).getEntries('searchResult')).toEqual([])
  })
})

describe('the settings command in the table of the commands of the editor', () => {
  it('has an icon name, the gear', () => {
    const found = createCommands(WORDS).find((one) => one.id === COMMAND.openSettings)

    expect(found?.iconName).toBe('settings')
  })
})
