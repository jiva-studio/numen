/**
 * The assembled window: the plugins of its target, loaded over HTTP and activated; the vaults the
 * user chooses from; and the tab the window opens on the chosen vault. The parts are made in the
 * order they depend on each other: the notices and the tabs first, then the services made from
 * them, then the host the services are given to.
 */
import { onUnmounted } from 'vue'
import {
  createPluginLoader,
  createWindowHost,
  usePluginStyles,
  type PluginLoaderDeps,
} from '@numen/plugin-loader'
import { createNoticeQueue, useWindowFrame } from '@numen/workbench'
import { createWindowAgent } from '@/entities/agent'
import { createWindowCommands } from '@/entities/plugin'
import { createWindowTabs } from '@/entities/tab'
import { createVaultsHost, useVaults } from '@/entities/vault'
import { useAppearance } from '@/features/appearance'
import { useAppStart } from '@/features/app-start'
import { loadWailsRuntime, useExternalLinks } from '@/features/external-links'
import { useWindowKeys } from '@/features/window-keys'
import { useWindowTasks } from '@/features/window-tasks'
import { formatError } from '@/shared/api/errors'
import { WINDOW_SOURCE } from '@/shared/lib/window'
import { createShellHost } from '../plugins/host'

/** What the window is given: how a plugin module is fetched and imported. */
export type WindowDeps = Pick<PluginLoaderDeps, 'fetch' | 'dynamicImport'>

export function createWindow(deps: WindowDeps = {}) {
  const notices = createNoticeQueue()
  const tabs = createWindowTabs()
  const frame = useWindowFrame(loadWailsRuntime)
  const windowAgent = createWindowAgent()
  const { vaults, isListing } = useVaults((text) => notices.show(WINDOW_SOURCE, text, 'error'))
  const { host, mountView, titleBarView } = createShellHost({
    tabs,
    notices,
    agent: windowAgent.agent,
    getPlatform: () => frame.platform.value,
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
    },
  })
  const loader = createPluginLoader({
    host,
    ...deps,
  })
  const start = useAppStart(loader)
  useWindowKeys({ tabs, vaults })
  useWindowTasks(notices)
  useExternalLinks(notices)
  const appearance = useAppearance(notices)

  onUnmounted(() => windowAgent.dispose())

  usePluginStyles(() => tabs.frontViews.value)

  return {
    tabs,
    vaults: { list: vaults, isListing },
    plugins: start,
    titleBarView,
    notices,
    mountView,
    appearance,
  }
}

/** What the component that draws the window is given. */
export type WindowParts = ReturnType<typeof createWindow>
