/**
 * The settings page of the plugin, opened through the command of the window.
 */
import { SETTINGS_OPEN_COMMAND, type Caught, type PluginContext } from '@numen/plugin-sdk'

/** Opens the settings page of the plugin; a window without commands opens nothing. */
export const createSettingsOpener = (context: Pick<PluginContext, 'plugin' | 'ui'>) => (): void => {
  context.ui.commands
    ?.run(SETTINGS_OPEN_COMMAND, undefined, undefined, { plugin: context.plugin.id })
    .catch((error: Caught) => context.ui.notices.show(String(error), 'error'))
}
