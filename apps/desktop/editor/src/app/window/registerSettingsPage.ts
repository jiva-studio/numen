/**
 * The settings of the window as a kind of tab, opened when a part asks for them.
 */
import type { PluginHost } from '@numen/plugin-loader'
import type { WindowFrame } from '@numen/workbench'
import type { SettingsStore } from '@/entities/settings'
import type { WindowTabs } from '@/entities/tab'
import { createSettingsPage } from '@/pages/settings'
import type { createWindowSettings } from '@/features/settings-commands'
import type { Emitter } from '@/shared/lib/emitter'
import type { MessageLog } from '@/shared/model/notices/messages'

/** What the settings page of the window is made from. */
export interface SettingsPageDeps {
  readonly rest: SettingsStore
  readonly windowSettings: Pick<ReturnType<typeof createWindowSettings>, 'appearance'>
  readonly log: MessageLog
  readonly windowTabs: WindowTabs
  readonly pluginHost: Pick<PluginHost, 'registered'>
  readonly frame: Pick<WindowFrame, 'platform'>
  readonly settingsRequests: Pick<Emitter<string | undefined>, 'subscribe'>
}

/** Makes the settings page, registers its kinds of tab, and opens it when it is asked for. */
export function registerSettingsPage(deps: SettingsPageDeps): void {
  const { pluginHost, frame, windowTabs } = deps
  const settingsPage = createSettingsPage({
    rest: deps.rest,
    appearance: deps.windowSettings.appearance,
    log: deps.log,
    windowTabs,
    getSettingsViews: (pluginId) => pluginHost.registered.getRegisteredSettingsViews(pluginId),
    getPlatform: () => frame.platform.value,
  })
  deps.settingsRequests.subscribe(settingsPage.configured.openSettings)
  windowTabs.groups.registerKinds(settingsPage.kinds)
}
