/**
 * The plugin entity's part of the answers: no plugin declares a command and no plugin reports
 * health, answered to the real clients over the router transport.
 */
import { COMMAND_ERROR_CODE } from '@numen/core-client'
import { pendingStream } from '../answers'

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
  pluginHealth: { watchHealth: pendingStream },
}
