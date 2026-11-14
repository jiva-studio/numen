/**
 * The runs of the commands the plugins declare, over a core that answers none of them, for a test
 * that runs no command or expects a run to be refused.
 */
import { createAnsweredApi } from './coreAnswers'

const { runDeclaredCommand, listDeclaredItemLists } = createAnsweredApi({}).declaredCommands

export const UNANSWERED_RUNS = {
  run: runDeclaredCommand,
  listItems: async (command: { readonly pluginId: string; readonly id: string }, path: string) => {
    await listDeclaredItemLists(path, [{ pluginId: command.pluginId, command: command.id }])
    return []
  },
}
