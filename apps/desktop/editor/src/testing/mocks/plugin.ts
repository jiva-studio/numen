/**
 * The plugin entity's part of the answers: the agent a plugin asks stays silent, and no plugin
 * declares a command, answered to the real clients over the router transport.
 */
import { COMMAND_ERROR_CODE } from '@numen/core-client'
import { pendingStream } from '../answers'

export const pluginAnswers = {
  agentService: {
    askAgent: pendingStream,
    finishConversation: () => ({}),
  },
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
