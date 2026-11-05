/**
 * The title bar of the window: drawn by the plugin that registers the view of the slot `titleBar`,
 * given the buttons the plugins place in it, and absent while no plugin draws it.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h, type PropType } from 'vue'
import { WorkspaceLayout } from '@numen/shell'
import type { PluginHost } from '@numen/plugin-host'
import { TitleBarHost } from '@/features/title-bar-view'
import {
  cards,
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
  isFullscreen: false,
  close: vi.fn(),
  minimise: vi.fn(),
  toggleMaximise: vi.fn(),
}))

vi.mock('../window/useWindowFrame', async () => {
  const { computed } = await import('vue')
  const { MAC_BUTTONS_INSET } = await import('@numen/shell')
  return {
    useWindowFrame: () => ({
      controls: computed(() => frame.controls),
      isMaximised: computed(() => frame.isMaximised),
      isFullscreen: computed(() => frame.isFullscreen),
      windowEdge: computed(() => ({
        leftInset: frame.controls === 'native-mac' && !frame.isFullscreen ? MAC_BUTTONS_INSET : 0,
        rightInset: 0,
      })),
      close: frame.close,
      minimise: frame.minimise,
      toggleMaximise: frame.toggleMaximise,
    }),
  }
})

/** The plugin host of the window last mounted, so a test registers a view as a plugin does. */
const hosts = vi.hoisted(() => ({ current: undefined as PluginHost | undefined }))

vi.mock('../plugins/useWindowPlugins', async (original) => {
  const module = await original<typeof import('../plugins/useWindowPlugins')>()
  return {
    ...module,
    createPluginSupport: (deps: Parameters<typeof module.createPluginSupport>[0]) => {
      const support = module.createPluginSupport(deps)
      hosts.current = support.pluginHost
      return support
    },
  }
})

const LEDGER = 'org.example.ledger'
const HARBOR = 'org.example.harbor'
const TIDE = 'org.example.tide'

/** A bar a plugin draws: the title and one button for each action and window control. */
const createBar = (name: string) =>
  defineComponent({
    name,
    props: {
      appName: { type: String, required: true },
      vaultName: { type: String, required: true },
      actions: { type: Array as PropType<readonly { id: string }[]>, required: true },
    },
    emits: ['run', 'choose', 'minimise', 'toggle-maximise', 'close'],
    setup:
      (props, { emit }) =>
      () =>
        h('header', { 'data-title-bar': name }, [
          h('span', { 'data-title': '' }, `${props.appName} — ${props.vaultName}`),
          ...props.actions.map((one) =>
            h('button', { 'data-title-bar-action': one.id, onClick: () => emit('run', one.id) }),
          ),
          h('button', { 'data-control': 'close', onClick: () => emit('close') }),
          h('button', { 'data-choice': 'trees', onClick: () => emit('choose', 'trees') }),
        ]),
  })

/** Registers the bar of a plugin as the plugin's frontend does when it is activated. */
const registerBar = async (pluginId: string, name = 'LedgerBar') => {
  hosts.current!.contextHost.registerSlotView(pluginId, 'titleBar', {
    id: 'bar',
    component: createBar(name),
  })
  await settle()
}

const placeTideChart = () => {
  declared.commands = [declareCommand(TIDE, 'chart.open', 'Open tide chart')]
  declared.menus = [
    { pluginId: TIDE, location: 'titleBar', command: 'chart.open', group: 'navigate' },
  ]
}

const getActions = (window: Awaited<ReturnType<typeof mountWindow>>) =>
  window.findComponent(TitleBarHost).props('actions') as readonly {
    id: string
    title: string
    group?: string
  }[]

const getWindowEdge = (window: Awaited<ReturnType<typeof mountWindow>>) =>
  window.findComponent(WorkspaceLayout).props('windowEdge') as
    { leftInset: number; rightInset: number } | undefined

afterEach(() => {
  frame.controls = 'windows'
  frame.isMaximised = false
  frame.isFullscreen = false
  frame.close.mockClear()
  hosts.current = undefined
})

describe('the window without a plugin that draws the title bar', () => {
  it('draws no bar, no window buttons and no drag area of its own', async () => {
    const window = await mountWindow()

    expect(window.find('[data-title-bar]').exists()).toBe(false)
    expect(window.find('[data-control]').exists()).toBe(false)
    expect(window.find('[data-window-controls]').exists()).toBe(false)
  })

  it('gives the workspace the top of the window, with no room kept on Windows', async () => {
    const window = await mountWindow()

    expect(getWindowEdge(window)).toStrictEqual({ leftInset: 0, rightInset: 0 })
    expect(window.find('main').element.children[0]?.hasAttribute('data-title-bar')).toBe(false)
  })

  it('keeps room for the macOS buttons in the top-left strip, and none in full screen', async () => {
    frame.controls = 'native-mac'
    const window = await mountWindow()

    expect(getWindowEdge(window)?.leftInset).toBe(80)

    frame.isFullscreen = true
    const fullscreen = await mountWindow()

    expect(getWindowEdge(fullscreen)?.leftInset).toBe(0)
  })
})

describe('the window with a plugin that draws the title bar', () => {
  it('draws the bar of the plugin above the workspace', async () => {
    const window = await mountWindow()

    await registerBar(LEDGER)

    expect(window.find('[data-title-bar="LedgerBar"]').exists()).toBe(true)
    expect(window.find('main').element.children[0]?.hasAttribute('data-title-bar')).toBe(true)
    expect(window.find('[data-title]').text()).toBe('numen — Physics')
  })

  it('moves the window by no strip of tabs and keeps no room in them', async () => {
    const window = await mountWindow()

    await registerBar(LEDGER)

    expect(getWindowEdge(window)).toBeUndefined()
    expect(window.find('[data-workspace-strip][data-window-drag]').exists()).toBe(false)
  })

  it('draws the bar of the plugin first by id, and reports the other plugin', async () => {
    const window = await mountWindow()

    await registerBar(LEDGER, 'LedgerBar')
    await registerBar(HARBOR, 'HarborBar')

    expect(window.find('[data-title-bar="HarborBar"]').exists()).toBe(true)
    expect(window.find('[data-title-bar="LedgerBar"]').exists()).toBe(false)
    expect(cards(window).join('\n')).toContain(HARBOR)
  })

  it('closes the window through the frame when the bar emits close', async () => {
    const window = await mountWindow()
    await registerBar(LEDGER)

    await window.find('[data-control="close"]').trigger('click')

    expect(frame.close).toHaveBeenCalledTimes(1)
  })

  it('passes the bar the state of the frame', async () => {
    frame.controls = 'native-mac'
    frame.isMaximised = true
    frame.isFullscreen = true
    const window = await mountWindow()

    const host = window.findComponent(TitleBarHost)
    expect(host.props('controls')).toBe('native-mac')
    expect(host.props('isMaximised')).toBe(true)
    expect(host.props('isFullscreen')).toBe(true)
  })

  it('opens the vault the bar chooses through the use case that opens a vault', async () => {
    listed.vaults.push({ id: 'trees', name: 'Trees', path: '/vaults/Trees', isMissing: false })
    const window = await mountWindow()
    await registerBar(LEDGER)

    await window.find('[data-choice="trees"]').trigger('click')
    await settle()

    expect(requests.opened).toStrictEqual(['trees'])
  })
})

describe('the buttons of the plugins in the title bar', () => {
  it('are the commands placed in titleBar, then the settings button', async () => {
    placeTideChart()

    const window = await mountWindow()

    expect(getActions(window).map((one) => one.id)).toStrictEqual([
      `${TIDE}:chart.open`,
      'app:settings.open',
    ])
    expect(getActions(window)[0]?.title).toBe('Open tide chart')
  })

  it('come in the groups and the order the plugins place them', async () => {
    declared.commands = [
      declareCommand(TIDE, 'chart.open', 'Open tide chart'),
      declareCommand(HARBOR, 'map.open', 'Open harbor map'),
      declareCommand(HARBOR, 'log.open', 'Open harbor log'),
    ]
    declared.menus = [
      { pluginId: HARBOR, location: 'titleBar', command: 'log.open', group: 'new', order: 1 },
      { pluginId: HARBOR, location: 'titleBar', command: 'map.open', group: 'navigate', order: 2 },
      { pluginId: TIDE, location: 'titleBar', command: 'chart.open', group: 'navigate', order: 1 },
    ]

    const window = await mountWindow()

    expect(getActions(window).map((one) => [one.id, one.group])).toStrictEqual([
      [`${TIDE}:chart.open`, 'navigate'],
      [`${HARBOR}:map.open`, 'navigate'],
      [`${HARBOR}:log.open`, 'new'],
      ['app:settings.open', 'settings'],
    ])
  })

  it('leave out a command placed in the search results', async () => {
    declared.commands = [declareCommand(TIDE, 'chart.open', 'Open tide chart')]
    declared.menus = [{ pluginId: TIDE, location: 'searchResult', command: 'chart.open' }]

    const window = await mountWindow()

    expect(getActions(window).map((one) => one.id)).toStrictEqual(['app:settings.open'])
  })

  it('are none while no vault is shown, even with commands placed', async () => {
    placeTideChart()
    listed.showing = ''

    const window = await mountWindow()

    expect(getActions(window)).toStrictEqual([])
  })

  it('open the palette at the name of a command that needs one when the bar runs it', async () => {
    declared.commands = []
    declared.menus = [{ pluginId: TIDE, location: 'titleBar', command: 'app:note.create' }]
    const window = await mountWindowWithPalette()
    await registerBar(LEDGER)
    const field = () => document.body.querySelector<HTMLInputElement>('[data-palette="field"]')

    await window.find('[data-title-bar-action="app:note.create"]').trigger('click')
    await settle()

    expect(field()).not.toBeNull()
    expect(requests.made).toStrictEqual([])
  })
})
