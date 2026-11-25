/**
 * The plugin entity's part of the answers: no plugin declares a command or contributes anything and no plugin reports
 * health, answered to the real clients over the router transport.
 */
import { WIRE_ERROR_CODE } from '@numen/core-client'
import { createPendingStream } from '../answers'

export const pluginAnswers = {
  pluginCommands: {
    listCommands: () => ({ commands: [] }),
    runPluginCommand: () => ({
      error: WIRE_ERROR_CODE.NOT_FOUND,
      message: 'no plugin declares it',
      openedPath: '',
    }),
  },
  contributions: {
    listContributions: () => ({ slots: [], handlers: [], keybindings: [], slotPoints: [] }),
    listContentTypes: () => ({ contentTypeByExtension: {} }),
  },
  pluginHealth: { watchHealth: createPendingStream },
}
