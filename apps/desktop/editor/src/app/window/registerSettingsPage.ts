/**
 * The settings of the window as a kind of tab, opened when a part asks for them.
 */
import type { Ref } from 'vue'
import type { CommandsHost, PluginHost } from '@numen/plugin-loader'
import type { WindowTabs } from '@/entities/tab'
import { createSettingsPage } from '@/pages/settings'
import type { createWindowSettings } from '@/features/settings-commands'
import type { PluginHealthItem } from '@/entities/plugin'
import type { createSlotGroups } from '@/features/plugin-host'
import type { createWindowFoundation } from './createWindowFoundation'

/** What the settings page reads of the foundation. */
type SettingsPageFoundation = Pick<
  ReturnType<typeof createWindowFoundation>,
  'api' | 'rest' | 'connections' | 'log' | 'frame' | 'settingsRequests'
>

/** What the settings page of the window is made from. */
export interface SettingsPageDeps {
  readonly foundation: SettingsPageFoundation
  readonly windowSettings: Pick<ReturnType<typeof createWindowSettings>, 'appearance'>
  readonly windowTabs: WindowTabs
  readonly plugins: SettingsPagePlugins
}

/** What the settings page reads of the plugins: their settings views, their health items and the commands they run. */
interface SettingsPagePlugins {
  readonly host: Pick<PluginHost, 'registered'>
  readonly health: Readonly<Ref<readonly PluginHealthItem[]>>
  readonly commands: Pick<CommandsHost, 'getIds' | 'run'>
  readonly getSlotGroups?: ReturnType<typeof createSlotGroups>
}

/** Makes the settings page, registers its kinds of tab, and opens it when it is asked for. */
export function registerSettingsPage(deps: SettingsPageDeps): void {
  const { foundation, plugins, windowTabs } = deps
  const { api, frame } = foundation
  const settingsPage = createSettingsPage({
    api: {
      settings: api.configuration.settings,
      license: api.configuration.license,
      identity: api.configuration.identity,
      keyFile: api.configuration.keyFile,
      listFiles: api.vault.file.listFiles,
      listCatalog: api.plugins.listCatalog,
      readReadme: api.plugins.readReadme,
      listRowOptions: api.configuration.connection.listRowOptions,
    },
    rest: foundation.rest,
    connections: foundation.connections,
    appearance: deps.windowSettings.appearance,
    log: foundation.log,
    windowTabs,
    getPlatform: () => frame.platform.value,
    plugins: {
      getSettingsViews: plugins.host.registered.getSettingsViews,
      ...(plugins.getSlotGroups ? { getSlotGroups: plugins.getSlotGroups } : {}),
      health: { items: plugins.health, commands: plugins.commands },
    },
  })
  foundation.settingsRequests.subscribe(settingsPage.configured.openSettings)
  windowTabs.kinds.registerKinds(settingsPage.kinds)
}
