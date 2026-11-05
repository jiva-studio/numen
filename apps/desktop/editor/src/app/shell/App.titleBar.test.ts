/**
 * The title bar the window draws above its workspace, and the buttons the plugins place in it.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { TitleBar } from '@numen/shell'
import {
  declareCommand,
  declared,
  listed,
  mountWindow,
  mountWindowWithPalette,
  requests,
  settle,
} from '@/testing/window'

const frame = vi.hoisted(() => ({
  controls: 'windows' as 'windows' | 'native-mac',
  isMaximised: false,
  close: vi.fn(),
  minimise: vi.fn(),
  toggleMaximise: vi.fn(),
}))

vi.mock('../window/useWindowFrame', async () => {
  const { computed } = await import('vue')
  return {
    useWindowFrame: () => ({
      controls: computed(() => frame.controls),
      isMaximised: computed(() => frame.isMaximised),
      close: frame.close,
      minimise: frame.minimise,
      toggleMaximise: frame.toggleMaximise,
    }),
  }
})

const TIDE = 'org.example.tide'
const REEF = 'org.example.reef'

/** A plugin that declares a command with an icon and places it in the title bar. */
const placeTideChart = () => {
  declared.commands = [declareCommand(TIDE, 'chart.open', 'Open tide chart')]
  declared.menus = [
    { pluginId: TIDE, location: 'titleBar', command: 'chart.open', group: 'navigate' },
  ]
}

const getActions = (window: Awaited<ReturnType<typeof mountWindow>>) =>
  window.findComponent(TitleBar).props('actions') as readonly {
    id: string
    title: string
    group?: string
  }[]

const getButton = (window: Awaited<ReturnType<typeof mountWindow>>, id: string) =>
  window.find(`[data-title-bar-action="${id}"]`)

describe('the title bar of the window', () => {
  it('is drawn above the workspace', async () => {
    const window = await mountWindow()

    const bar = window.find('[data-title-bar]')
    expect(bar.exists()).toBe(true)
    const order = window.find('main').element.children
    expect(order[0]?.hasAttribute('data-title-bar')).toBe(true)
  })

  it('names the application and the vault', async () => {
    const window = await mountWindow()

    expect(window.find('[data-title]').text()).toBe('numen — Physics')
  })

  it('names the application alone where no vault is open', async () => {
    listed.showing = ''

    const window = await mountWindow()

    expect(window.find('[data-title]').text()).toBe('numen')
  })

  it('draws the window controls and the settings button where no plugin places a command', async () => {
    const window = await mountWindow()

    const bar = window.findComponent(TitleBar)
    expect(bar.props('controls')).toBe('windows')
    expect(getActions(window).map((one) => one.id)).toStrictEqual(['app:settings.open'])
    expect(window.findAll('[data-title-bar] button')).toHaveLength(4)
  })
})

describe('the buttons of the plugins in the title bar', () => {
  it('draws one button for the command a plugin places, then the settings button, next to the three window controls', async () => {
    placeTideChart()

    const window = await mountWindow()

    expect(getActions(window).map((one) => one.id)).toStrictEqual([
      `${TIDE}:chart.open`,
      'app:settings.open',
    ])
    expect(getActions(window)[0]?.title).toBe('Open tide chart')
    expect(window.findAll('[data-title-bar] button')).toHaveLength(5)
    expect(getButton(window, `${TIDE}:chart.open`).exists()).toBe(true)
  })

  it('draws the buttons in the groups and the order the plugins place them', async () => {
    declared.commands = [
      declareCommand(TIDE, 'chart.open', 'Open tide chart'),
      declareCommand(REEF, 'map.open', 'Open reef map'),
      declareCommand(REEF, 'log.open', 'Open reef log'),
    ]
    declared.menus = [
      { pluginId: REEF, location: 'titleBar', command: 'log.open', group: 'new', order: 1 },
      { pluginId: REEF, location: 'titleBar', command: 'map.open', group: 'navigate', order: 2 },
      { pluginId: TIDE, location: 'titleBar', command: 'chart.open', group: 'navigate', order: 1 },
    ]

    const window = await mountWindow()

    expect(getActions(window).map((one) => [one.id, one.group])).toStrictEqual([
      [`${TIDE}:chart.open`, 'navigate'],
      [`${REEF}:map.open`, 'navigate'],
      [`${REEF}:log.open`, 'new'],
      ['app:settings.open', 'settings'],
    ])
  })

  it('draws no button for a command placed in the search results, and the settings button alone', async () => {
    declared.commands = [declareCommand(TIDE, 'chart.open', 'Open tide chart')]
    declared.menus = [{ pluginId: TIDE, location: 'searchResult', command: 'chart.open' }]

    const window = await mountWindow()

    expect(getActions(window).map((one) => one.id)).toStrictEqual(['app:settings.open'])
  })

  it('draws no button while no vault is shown, even with commands placed', async () => {
    placeTideChart()
    listed.showing = ''

    const window = await mountWindow()

    expect(window.find('[data-title]').text()).toBe('numen')
    expect(getActions(window)).toStrictEqual([])
    expect(window.findAll('[data-title-bar] button')).toHaveLength(3)
  })

  it('draws no pressed state on a button after it is pressed', async () => {
    placeTideChart()
    const window = await mountWindow()
    const button = getButton(window, `${TIDE}:chart.open`)

    await button.trigger('click')
    await settle()

    for (const name of ['aria-pressed', 'aria-checked', 'data-pressed', 'data-active']) {
      expect(getButton(window, `${TIDE}:chart.open`).attributes(name)).toBeUndefined()
    }
  })

  it('draws the same buttons in a window detachedWindow off from a tab', async () => {
    declared.commands = [
      declareCommand(TIDE, 'chart.open', 'Open tide chart'),
      declareCommand(REEF, 'map.open', 'Open reef map'),
    ]
    declared.menus = [
      { pluginId: TIDE, location: 'titleBar', command: 'chart.open', group: 'navigate', order: 1 },
      { pluginId: REEF, location: 'titleBar', command: 'map.open', group: 'new', order: 1 },
    ]
    const main = await mountWindow()
    const inMain = getActions(main).map((one) => [one.id, one.title, one.group])
    expect(inMain.map((one) => one[0])).toStrictEqual([
      `${TIDE}:chart.open`,
      `${REEF}:map.open`,
      'app:settings.open',
    ])
    const address = globalThis.location.href

    globalThis.history.replaceState(null, '', '/?window=secondary')
    try {
      const detachedWindow = await mountWindow()

      expect(getActions(detachedWindow).map((one) => [one.id, one.title, one.group])).toStrictEqual(
        inMain,
      )
    } finally {
      globalThis.history.replaceState(null, '', address)
    }
  })
})

describe('a press on a button of the title bar', () => {
  it('opens the palette at the name of a command that needs one, and makes nothing before it is typed', async () => {
    declared.commands = []
    declared.menus = [{ pluginId: TIDE, location: 'titleBar', command: 'app:note.create' }]
    const window = await mountWindowWithPalette()
    const field = () => document.body.querySelector<HTMLInputElement>('[data-palette="field"]')

    await getButton(window, 'app:note.create').trigger('click')
    await settle()

    expect(field()).not.toBeNull()
    expect(requests.made).toStrictEqual([])

    field()!.value = 'Brine'
    field()!.dispatchEvent(new Event('input'))
    await settle()
    field()!.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }))
    await settle()

    expect(requests.made).toStrictEqual(['Brine'])
  })
})

describe('the window controls of the title bar with buttons placed', () => {
  it('stay the circles of the system on macOS and draw none of its own', async () => {
    placeTideChart()
    frame.controls = 'native-mac'

    const window = await mountWindow()

    expect(window.findComponent(TitleBar).props('controls')).toBe('native-mac')
    expect(window.findAll('[data-control]')).toHaveLength(0)
    expect(getActions(window).map((one) => one.id)).toStrictEqual([
      `${TIDE}:chart.open`,
      'app:settings.open',
    ])
  })

  it('close the window through the frame, whatever the buttons placed', async () => {
    placeTideChart()
    frame.controls = 'windows'
    frame.close.mockClear()
    const window = await mountWindow()

    await window.find('[data-control="close"]').trigger('click')

    expect(frame.close).toHaveBeenCalledTimes(1)
  })

  it('follow the maximised state of the frame and not the buttons', async () => {
    placeTideChart()
    frame.controls = 'windows'
    frame.isMaximised = true

    const window = await mountWindow()

    expect(window.findComponent(TitleBar).props('isMaximised')).toBe(true)
    frame.isMaximised = false
  })
})

afterEach(() => {
  frame.controls = 'windows'
  frame.isMaximised = false
  frame.close.mockClear()
})
