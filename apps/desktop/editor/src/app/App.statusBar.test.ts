/**
 * The bar at the bottom of the window: drawn by the plugin that registers the view of the slot
 * `statusBar`, and absent while no plugin draws it. The tasks of the window reach the plugins as
 * their background tasks.
 */
import { describe, expect, it } from 'vitest'
import { defineComponent, h } from 'vue'
import type { BackgroundTask, PluginHost } from '@numen/plugin-loader'
import { WorkspaceLayout } from '@numen/workbench'
import {
  getWindowParts,
  mountWindow as mountPlainWindow,
  settle,
  vaultState,
} from '@/testing/window'

/** The plugin host of the window last mounted, so a test registers a view as a plugin does. */
const hosts = { current: undefined as PluginHost | undefined }

const mountWindow = async () => {
  const window = await mountPlainWindow()
  hosts.current = getWindowParts(window).pluginHost
  return window
}

const LANTERN = 'org.example.lantern'

const LanternBar = defineComponent({
  name: 'LanternBar',
  setup: () => () => h('footer', { 'data-lantern-bar': '' }, 'lantern'),
})

const registerBar = async () => {
  hosts.current!.contextHost.views.registerSlotView(LANTERN, 'statusBar', {
    id: 'bar',
    component: LanternBar,
  })
  await settle()
}

describe('the status bar of the window', () => {
  it('is not drawn while no plugin draws it', async () => {
    const window = await mountWindow()

    expect(window.find('[data-lantern-bar]').exists()).toBe(false)
    expect(window.find('[role="region"][aria-label="Status Bar"]').exists()).toBe(false)
    expect(window.find('.status-bar').exists()).toBe(false)
  })

  it('is the view of the plugin that fills the slot, drawn after the workspace', async () => {
    const window = await mountWindow()

    await registerBar()

    const bar = window.find('[data-lantern-bar]')
    expect(bar.text()).toBe('lantern')
    const workspace = window.findComponent(WorkspaceLayout).element as Node
    expect(workspace.compareDocumentPosition(bar.element) & Node.DOCUMENT_POSITION_FOLLOWING).toBe(
      Node.DOCUMENT_POSITION_FOLLOWING,
    )
  })

  it('goes when the plugin takes its view away', async () => {
    const window = await mountWindow()
    await registerBar()

    hosts.current!.lifecycle.disposePlugin(LANTERN)
    await settle()

    expect(window.find('[data-lantern-bar]').exists()).toBe(false)
  })
})

describe('the background tasks a plugin is given', () => {
  it('list the failed work of the window', async () => {
    vaultState.opening = null
    vaultState.error = 'the vault folder is not there'
    await mountWindow()
    const host = hosts.current!.contextHost.window.getBackgroundTasks()

    let tasks: readonly BackgroundTask[] = []
    host!.watch((list) => (tasks = list)).dispose()

    expect(tasks.filter((one) => one.state === 'failed').map((one) => one.about)).toStrictEqual([
      'the vault folder is not there',
    ])
  })
})
