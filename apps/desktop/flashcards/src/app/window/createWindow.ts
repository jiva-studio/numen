/**
 * The assembled window: the plugins of its target, loaded over HTTP and activated; the vaults the
 * user chooses from; and the tab the window opens on the chosen vault. The parts are made in the
 * order they depend on each other: the clients over the transport the window is given, the notices
 * and the tabs first, then the services made from them, then the host the services are given to.
 */
import {
  createPluginLoader,
  createWindowHost,
  createFrameRuntimeLoader,
  createWindowFrameControls,
  openExternalUrl,
  usePluginStyles,
  type PluginLoaderDeps,
} from '@numen/plugin-loader'
import { createNoticeQueue, useWindowFrame } from '@numen/workbench'
import {
  createPluginConnectionFinder,
  createPluginLister,
  createWindowCommands,
} from '@/entities/plugin'
import { createWindowTabs } from '@/entities/tab'
import {
  createVaultLister,
  createVaultsHost,
  createWindowRelations,
  useVaults,
} from '@/entities/vault'
import { createThemes, useAppearance } from '@/features/appearance'
import { useAppStart } from '@/features/app-start'
import { useSlotLayouts } from '@/features/slot-layouts'
import { useWindowKeys } from '@/features/window-keys'
import { createTaskApi, useWindowTasks } from '@/features/window-tasks'
import { createWindowClients, type Transport } from '@/shared/api/clients'
import { formatError } from '@/shared/api/errors'
import { WINDOW_SOURCE } from '@/shared/lib/window'
import { createShellHost } from '../plugins/host'

/** What the window is given: how a plugin module is fetched and imported. */
export type WindowDeps = Pick<PluginLoaderDeps, 'fetch' | 'dynamicImport'>

export function createWindow(transport: Transport, deps: WindowDeps = {}) {
  const clients = createWindowClients(transport)
  const listPlugins = createPluginLister(clients.pluginService)
  const notices = createNoticeQueue()
  const tabs = createWindowTabs()
  const frameRuntime = createFrameRuntimeLoader()
  const frame = useWindowFrame(createWindowFrameControls(frameRuntime))
  const { vaults, isListing } = useVaults(createVaultLister(clients.vaultsService), (text) =>
    notices.show(WINDOW_SOURCE, text, 'error'),
  )
  const backgroundTasks = useWindowTasks(createTaskApi(clients.windowService), notices)
  const { getSlotLayout } = useSlotLayouts(clients.settingsService)
  const { host, styles, mountView, titleBarView } = createShellHost({
    openExternal: (href) => openExternalUrl(href, frameRuntime),
    relations: createWindowRelations(clients.notes),
    getConnections: createPluginConnectionFinder(clients.connectionCalls),
    tabs,
    notices,
    getPlatform: () => frame.platform.value,
    getSlotLayout,
    services: {
      commands: createWindowCommands(),
      vaults: createVaultsHost({
        list: vaults,
        getTabVault: () => tabs.tabVault.value,
        openTab: tabs.openTab,
      }),
      window: createWindowHost(frame, (error) =>
        notices.show(WINDOW_SOURCE, formatError(error), 'error'),
      ),
      backgroundTasks,
    },
  })
  const loader = createPluginLoader({
    host,
    styles,
    ...deps,
  })
  const start = useAppStart(listPlugins, loader)
  useWindowKeys({ tabs, vaults })
  const appearance = useAppearance({ themes: createThemes(clients.theme), listPlugins }, notices)

  usePluginStyles(styles, () => tabs.frontPlugins.value)

  return {
    tabs,
    vaults: { list: vaults, isListing },
    plugins: start,
    views: { titleBar: titleBarView, styles },
    notices,
    backgroundTasks,
    mountView,
    appearance,
  }
}

/** What the component that draws the window is given. */
export type WindowParts = ReturnType<typeof createWindow>
