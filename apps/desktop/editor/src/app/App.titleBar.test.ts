/**
 * The title bar of the window: drawn by the plugin that registers the view of the slot `titleBar`,
 * which reads the vaults and the window from its context, and absent while no plugin draws it.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h, shallowRef } from 'vue'
import { WorkspaceLayout } from '@numen/workbench'
import type { PluginHost, VaultsHost, WindowHost } from '@numen/plugin-loader'
import {
  cards,
  getWindowParts,
  listed,
  mountWindow as mountPlainWindow,
  requests,
  settle,
} from '@/testing/window'
import { COMMAND } from '@/shared/model/command'
import type * as ShellModule from '@numen/workbench'

const frame = vi.hoisted(() => ({
  platform: 'windows' as 'windows' | 'darwin',
  isMaximized: false,
  isFullscreen: false,
  closeWindow: vi.fn(async () => undefined),
  minimizeWindow: vi.fn(async () => undefined),
  toggleMaximizeWindow: vi.fn(async () => undefined),
}))

vi.mock('@numen/workbench', async (importOriginal) => {
  const { computed } = await import('vue')
  const shell = await importOriginal<typeof ShellModule>()
  const { MAC_BUTTONS_INSET } = shell
  return {
    ...shell,
    useWindowFrame: () => ({
      platform: computed(() => frame.platform),
      isMaximized: computed(() => frame.isMaximized),
      isFullscreen: computed(() => frame.isFullscreen),
      windowEdge: computed(() => ({
        leftInset: frame.platform === 'darwin' && !frame.isFullscreen ? MAC_BUTTONS_INSET : 0,
        rightInset: 0,
      })),
      closeWindow: frame.closeWindow,
      minimizeWindow: frame.minimizeWindow,
      toggleMaximizeWindow: frame.toggleMaximizeWindow,
    }),
  }
})

/** The plugin host of the window last mounted, so a test registers a view as a plugin does. */
const hosts = { current: undefined as PluginHost | undefined }

const mountWindow = async () => {
  const window = await mountPlainWindow()
  hosts.current = getWindowParts(window).pluginHost
  return window
}

const LEDGER = 'org.example.ledger'
const HARBOR = 'org.example.harbor'

/** A bar a plugin draws from the vaults, the window and the app its context holds. */
interface BarContext {
  readonly app: { readonly name: string }
  readonly vaults: VaultsHost | undefined
  readonly window: WindowHost | undefined
}

const createBar = (name: string, context: BarContext) =>
  defineComponent({
    name,
    setup: () => {
      const vaults = shallowRef(context.vaults!.getState())
      const window = shallowRef(context.window!.getState())
      context.vaults!.watch((state) => (vaults.value = state))
      context.window!.watch((state) => (window.value = state))
      return () =>
        h(
          'header',
          { 'data-title-bar': name, 'data-maximized': String(window.value.isMaximized) },
          [
            h('span', { 'data-title': '' }, `${context.app.name} — ${vaults.value.current?.name}`),
            h('span', { 'data-vaults': '' }, vaults.value.vaults.map((one) => one.id).join(',')),
            h('button', { 'data-control': 'close', onClick: () => context.window!.close() }),
            h('button', {
              'data-vault': 'trees',
              onClick: () => context.vaults!.switchTo('trees'),
            }),
          ],
        )
    },
  })

const registerBar = async (pluginId: string, name = 'LedgerBar') => {
  const { contextHost } = hosts.current!
  const context = {
    app: contextHost.info.app,
    vaults: contextHost.vault.getVaults(),
    window: contextHost.window.getWindow(),
  }
  contextHost.views.registerSlotView(pluginId, 'titleBar', {
    id: 'bar',
    component: createBar(name, context),
  })
  await settle()
}

const getWindowEdge = (window: Awaited<ReturnType<typeof mountWindow>>) =>
  window.findComponent(WorkspaceLayout).props('windowEdge') as
    { leftInset: number; rightInset: number } | undefined

afterEach(() => {
  frame.platform = 'windows'
  frame.isMaximized = false
  frame.isFullscreen = false
  frame.closeWindow.mockClear()
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
    frame.platform = 'darwin'
    const window = await mountWindow()

    expect(getWindowEdge(window)?.leftInset).toBe(80)

    frame.isFullscreen = true
    const fullscreen = await mountWindow()

    expect(getWindowEdge(fullscreen)?.leftInset).toBe(0)
  })
})

describe('the window with a plugin that draws the title bar', () => {
  it('draws the bar of the plugin above the workspace, titled from the context', async () => {
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
    expect(cards().join('\n')).toContain(HARBOR)
  })

  it('closes the window through the frame when the bar closes the window of its context', async () => {
    const window = await mountWindow()
    await registerBar(LEDGER)

    await window.find('[data-control="close"]').trigger('click')

    expect(frame.closeWindow).toHaveBeenCalledTimes(1)
  })

  it('gives the bar the state of the frame', async () => {
    frame.isMaximized = true
    const window = await mountWindow()

    await registerBar(LEDGER)

    expect(window.find('[data-title-bar]').attributes('data-maximized')).toBe('true')
  })

  it('gives the bar only the vaults of the installation', async () => {
    listed.vaults.push({ id: 'trees', name: 'Trees', path: '/vaults/Trees', isMissing: false })
    const window = await mountWindow()

    await registerBar(LEDGER)

    const ids = window.find('[data-vaults]').text().split(',')
    expect(ids.at(-1)).toBe('trees')
    expect(ids).not.toContain(COMMAND.createVault)
  })

  it('opens the vault the bar switches to through the use case that opens a vault', async () => {
    listed.vaults.push({ id: 'trees', name: 'Trees', path: '/vaults/Trees', isMissing: false })
    const window = await mountWindow()
    await registerBar(LEDGER)

    await window.find('[data-vault="trees"]').trigger('click')
    await settle()

    expect(requests.opened).toStrictEqual(['trees'])
  })
})

describe('the platform the plugins of the window are given', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it.each([
    ['windows', 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15'],
    ['darwin', 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/605.1.15'],
  ] as const)(
    'is %s where the frame reports it and the user agent names another',
    async (platform, agent) => {
      vi.stubGlobal('navigator', { ...navigator, userAgent: agent })
      frame.platform = platform
      await mountWindow()

      expect(hosts.current!.contextHost.info.app.platform).toBe(platform)
    },
  )
})
