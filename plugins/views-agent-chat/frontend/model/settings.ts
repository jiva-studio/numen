/**
 * The settings page of the plugin, opened through the command of the window.
 */
import {
  SETTINGS_OPEN_COMMAND,
  type Caught,
  type HostServices,
  type PluginContext,
} from '@numen/plugin-sdk'

/** Opens the settings page of the plugin; a window without commands opens nothing. */
export const createSettingsOpener =
  (plugin: Pick<PluginContext, 'identity'>, host: Pick<HostServices, 'ui'>) => (): void => {
    host.ui.commands
      ?.run(SETTINGS_OPEN_COMMAND, undefined, undefined, { plugin: plugin.identity.id })
      .catch((error: Caught) => host.ui.notices.show(String(error), 'error'))
  }
