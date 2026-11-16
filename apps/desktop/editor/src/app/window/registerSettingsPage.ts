/**
 * The settings of the window as a kind of tab, opened when a part asks for them.
 */
import type { Ref } from 'vue'
import type { CommandsHost, PluginHost } from '@numen/plugin-loader'
import type { WindowTabs } from '@/entities/tab'
import { createSettingsPage } from '@/pages/settings'
import type { createWindowSettings } from '@/features/settings-commands'
import type { PluginHealthItem } from '@/entities/plugin'
import type { WindowApi } from './createWindowApi'
import type { WindowFoundation } from './createWindowFoundation'

/** What the settings page of the window is made from. */
export interface SettingsPageDeps {
  readonly api: WindowApi
  readonly rest: WindowFoundation['rest']
  readonly connections: WindowFoundation['connections']
  readonly log: WindowFoundation['log']
  readonly frame: Pick<WindowFoundation['frame'], 'platform'>
  readonly settingsRequests: Pick<WindowFoundation['settingsRequests'], 'subscribe'>
  readonly windowSettings: Pick<ReturnType<typeof createWindowSettings>, 'appearance'>
  readonly windowTabs: WindowTabs
  readonly plugins: SettingsPagePlugins
}

/** What the settings page reads of the plugins: their settings views, their health items and the commands they run. */
interface SettingsPagePlugins {
  readonly host: Pick<PluginHost, 'registered'>
  readonly health: Readonly<Ref<readonly PluginHealthItem[]>>
  readonly commands: Pick<CommandsHost, 'getIds' | 'run'>
}

/** Makes the settings page, registers its kinds of tab, and opens it when it is asked for. */
export function registerSettingsPage(deps: SettingsPageDeps): void {
  const { api, frame, plugins, windowTabs } = deps
  const settingsPage = createSettingsPage({
    api: {
      settings: api.configuration.settings,
      listFiles: api.vault.file.listFiles,
      listCatalog: api.plugins.listCatalog,
      listRowOptions: api.configuration.connection.listRowOptions,
    },
    rest: deps.rest,
    connections: deps.connections,
    appearance: deps.windowSettings.appearance,
    log: deps.log,
    windowTabs,
    getPlatform: () => frame.platform.value,
    plugins: {
      getSettingsViews: plugins.host.registered.getSettingsViews,
      health: { items: plugins.health, commands: plugins.commands },
    },
  })
  deps.settingsRequests.subscribe(settingsPage.configured.openSettings)
  windowTabs.kinds.registerKinds(settingsPage.kinds)
}
