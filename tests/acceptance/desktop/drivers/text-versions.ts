/** The text versions of a file (the text layer, a recognition, a proofreading), as the host lists and deletes them. */
import type { APIRequestContext } from 'playwright/test'
import { callRPC } from '../../../e2e/desktop/tests/support/vault.js'

const DELETE_COMMAND = 'app:version.delete'

interface CommandItemLists {
  lists?: { items?: { id: string }[] }[]
}

export const listVersionIds = async (request: APIRequestContext, filePath: string): Promise<string[]> => {
  const answer = (await callRPC(request, 'CommandService/ListCommandItems', {
    path: filePath,
    commands: [{ command: DELETE_COMMAND }],
  })) as CommandItemLists
  return (answer.lists ?? []).flatMap((list) => (list.items ?? []).map((item) => item.id))
}

export const deleteVersion = async (request: APIRequestContext, filePath: string, id: string): Promise<void> => {
  await callRPC(request, 'CommandService/RunPluginCommand', { command: DELETE_COMMAND, path: filePath, itemId: id })
}
