/**
 * Frontend entrypoint for the Title bar plugin: the bar at the top of each window, drawn in the
 * slot `titleBar`. The bar reads its slots, the vaults and the window from the host services.
 */
import { defineComponent, h, shallowRef } from 'vue'
import { runInBackground } from '@numen/ui'
import { definePlugin, type HostServices, type PluginContext, type Caught } from '@numen/plugin-sdk'
import {
  BUTTONS_SETTING,
  HIDE_IN_FULLSCREEN_SETTING,
  isBarShown,
  readButtonsSetting,
  readHideInFullscreenSetting,
} from './lib/frame'
import { useTitleBarHosts } from './model/useTitleBarHosts'
import TitleBar from './ui/TitleBar.vue'

export { TitleBar }

/** The id the manifest declares the view of the title bar under. */
export const TITLE_BAR_VIEW = 'bar'

/** The command of the window that creates a vault, which the last row of the list of vaults runs. */
const CREATE_VAULT = 'app:vault.create'

/** The view of the slot: the bar, with the window buttons where `buttons` says. */
export function createTitleBarView(
  plugin: Pick<PluginContext, 'identity' | 'settings'>,
  host: Pick<HostServices, 'ui' | 'vault'>,
) {
  const buttons = shallowRef(readButtonsSetting(plugin.settings.get(BUTTONS_SETTING)))
  plugin.settings.onChange(BUTTONS_SETTING, (value) => {
    buttons.value = readButtonsSetting(value)
  })
  const hideInFullscreen = shallowRef(
    readHideInFullscreenSetting(plugin.settings.get(HIDE_IN_FULLSCREEN_SETTING)),
  )
  plugin.settings.onChange(HIDE_IN_FULLSCREEN_SETTING, (value) => {
    hideInFullscreen.value = readHideInFullscreenSetting(value)
  })
  const reportError = (error: Caught): void =>
    host.ui.notices.show(error instanceof Error ? error.message : String(error), 'error')
  return defineComponent({
    name: 'TitleBarView',
    setup: () => {
      const hosts = useTitleBarHosts(host)
      return () =>
        !isBarShown(hideInFullscreen.value, hosts.window.value.isFullscreen)
          ? null
          : h(TitleBar, {
              appName: plugin.identity.app.name,
              platform: plugin.identity.app.platform,
              buttons: buttons.value,
              isMaximized: hosts.window.value.isMaximized,
              isFullscreen: hosts.window.value.isFullscreen,
              vaults: hosts.vaults.value,
              commands: hosts.commands.value,
              status: hosts.status.value,
              canCreateVault: hosts.commandIds.value.includes(CREATE_VAULT),
              listItems: (command: string) =>
                host.ui.commands?.listItems(command) ?? Promise.resolve([]),
              reportError,
              onRunCommand: (
                command: string,
                itemId?: string,
                values?: Readonly<Record<string, string>>,
              ) => {
                if (host.ui.commands) {
                  runInBackground(
                    host.ui.commands.run(command, undefined, itemId, values),
                    reportError,
                  )
                }
              },
              onSwitchVault: (id: string) => {
                if (host.vault.vaults) runInBackground(host.vault.vaults.switchTo(id), reportError)
              },
              onCreateVault: () => {
                if (host.ui.commands)
                  runInBackground(host.ui.commands.run(CREATE_VAULT), reportError)
              },
              onMinimizeWindow: () => host.ui.window?.minimize(),
              onToggleMaximizeWindow: () => host.ui.window?.toggleMaximize(),
              onCloseWindow: () => host.ui.window?.close(),
            })
    },
  })
}

export const activate = (plugin: PluginContext, host: HostServices): void => {
  plugin.register.slotView({ id: TITLE_BAR_VIEW, component: createTitleBarView(plugin, host) })
}

export default definePlugin({
  activate,
})
