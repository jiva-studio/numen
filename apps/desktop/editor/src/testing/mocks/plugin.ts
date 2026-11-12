/**
 * The plugin entity's part of the answers: no plugin declares a command, answered to the real
 * clients over the router transport.
 */
import { COMMAND_ERROR_CODE } from '@numen/core-client'

export const pluginAnswers = {
  pluginCommands: {
    listCommands: () => ({
      commands: [],
      slots: [],
      handlers: [],
      keybindings: [],
      contentTypeByExtension: {},
    }),
    runPluginCommand: () => ({
      error: COMMAND_ERROR_CODE.UNKNOWN_COMMAND,
      message: 'no plugin declares it',
      openedPath: '',
    }),
  },
}
