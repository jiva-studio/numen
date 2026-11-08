/**
 * Frontend entrypoint for the Title bar plugin: the bar at the top of each window, drawn in the
 * slot `titleBar`. The bar reads its slots, the vaults and the window from the context.
 */
import { defineComponent, h, shallowRef } from 'vue'
import { runInBackground } from '@numen/ui'
import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import { BUTTONS_SETTING, readButtonsSetting } from './lib/frame'
import { useTitleBarHosts } from './model/useTitleBarHosts'
import TitleBar from './ui/TitleBar.vue'

export { TitleBar }

/** The id the manifest declares the view of the title bar under. */
export const TITLE_BAR_VIEW = 'bar'

/** The command of the window that creates a vault, which the last row of the list of vaults runs. */
const CREATE_VAULT = 'app:vault.create'

/** The view of the slot: the bar over the context, with the window buttons where `buttons` says. */
export function createTitleBarView(context: PluginContext) {
  const buttons = shallowRef(readButtonsSetting(context.getSetting(BUTTONS_SETTING)))
  context.onSettingChange(BUTTONS_SETTING, (value) => {
    buttons.value = readButtonsSetting(value)
  })
  const reportError = (error: unknown): void =>
    context.notices?.show(error instanceof Error ? error.message : String(error), 'error')
  return defineComponent({
    name: 'TitleBarView',
    setup: () => {
      const hosts = useTitleBarHosts(context)
      return () =>
        h(TitleBar, {
          appName: context.app.name,
          platform: context.app.platform,
          buttons: buttons.value,
          isMaximized: hosts.window.value.isMaximized,
          isFullscreen: hosts.window.value.isFullscreen,
          vaults: hosts.vaults.value,
          commands: hosts.commands.value,
          status: hosts.status.value,
          canCreateVault: hosts.commandIds.value.includes(CREATE_VAULT),
          listItems: (command: string) =>
            context.commands?.listItems(command) ?? Promise.resolve([]),
          reportError,
          onRunCommand: (command: string, argument?: string) => {
            if (context.commands) {
              runInBackground(context.commands.run(command, undefined, argument), reportError)
            }
          },
          onSwitchVault: (id: string) => {
            if (context.vaults) runInBackground(context.vaults.switchTo(id), reportError)
          },
          onCreateVault: () => {
            if (context.commands) runInBackground(context.commands.run(CREATE_VAULT), reportError)
          },
          onMinimizeWindow: () => context.window?.minimize(),
          onToggleMaximizeWindow: () => context.window?.toggleMaximize(),
          onCloseWindow: () => context.window?.close(),
        })
    },
  })
}

export const activate = (context: PluginContext): void => {
  context.registerSlotView({ id: TITLE_BAR_VIEW, component: createTitleBarView(context) })
}

export default definePlugin({
  activate,
})
