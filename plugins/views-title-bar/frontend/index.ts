/**
 * Frontend entrypoint for the Title bar plugin: the bar at the top of each window, drawn in the
 * slot `titleBar`. The bar reads its slots, the vaults and the window from the context.
 */
import { defineComponent, h, shallowRef } from 'vue'
import { runInBackground } from '@numen/ui'
import { definePlugin, type PluginContext, type Caught } from '@numen/plugin-sdk'
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

/** The view of the slot: the bar over the context, with the window buttons where `buttons` says. */
export function createTitleBarView(
  context: Pick<PluginContext, 'plugin' | 'settings' | 'ui' | 'vault'>,
) {
  const buttons = shallowRef(readButtonsSetting(context.settings.get(BUTTONS_SETTING)))
  context.settings.onChange(BUTTONS_SETTING, (value) => {
    buttons.value = readButtonsSetting(value)
  })
  const hideInFullscreen = shallowRef(
    readHideInFullscreenSetting(context.settings.get(HIDE_IN_FULLSCREEN_SETTING)),
  )
  context.settings.onChange(HIDE_IN_FULLSCREEN_SETTING, (value) => {
    hideInFullscreen.value = readHideInFullscreenSetting(value)
  })
  const reportError = (error: Caught): void =>
    context.ui.notices.show(error instanceof Error ? error.message : String(error), 'error')
  return defineComponent({
    name: 'TitleBarView',
    setup: () => {
      const hosts = useTitleBarHosts(context)
      return () =>
        !isBarShown(hideInFullscreen.value, hosts.window.value.isFullscreen)
          ? null
          : h(TitleBar, {
              appName: context.plugin.app.name,
              platform: context.plugin.app.platform,
              buttons: buttons.value,
              isMaximized: hosts.window.value.isMaximized,
              isFullscreen: hosts.window.value.isFullscreen,
              vaults: hosts.vaults.value,
              commands: hosts.commands.value,
              status: hosts.status.value,
              canCreateVault: hosts.commandIds.value.includes(CREATE_VAULT),
              listItems: (command: string) =>
                context.ui.commands?.listItems(command) ?? Promise.resolve([]),
              reportError,
              onRunCommand: (
                command: string,
                itemId?: string,
                values?: Readonly<Record<string, string>>,
              ) => {
                if (context.ui.commands) {
                  runInBackground(
                    context.ui.commands.run(command, undefined, itemId, values),
                    reportError,
                  )
                }
              },
              onSwitchVault: (id: string) => {
                if (context.vault.vaults)
                  runInBackground(context.vault.vaults.switchTo(id), reportError)
              },
              onCreateVault: () => {
                if (context.ui.commands)
                  runInBackground(context.ui.commands.run(CREATE_VAULT), reportError)
              },
              onMinimizeWindow: () => context.ui.window?.minimize(),
              onToggleMaximizeWindow: () => context.ui.window?.toggleMaximize(),
              onCloseWindow: () => context.ui.window?.close(),
            })
    },
  })
}

export const activate = (context: PluginContext): void => {
  context.register.slotView({ id: TITLE_BAR_VIEW, component: createTitleBarView(context) })
}

export default definePlugin({
  activate,
})
