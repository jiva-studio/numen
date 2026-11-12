/** The text versions of a file (the text layer, a recognition, a proofreading), as the host lists and deletes them. */
import type { APIRequestContext } from 'playwright/test'
import { callRPC } from '../../../e2e/desktop/tests/support/vault.js'

const DELETE_COMMAND = 'app:version.delete'

/** The title of the version a file has before any plugin makes text of it. */
const TEXT_LAYER = 'Text layer'

export interface TextVersion {
  id: string
  title: string
}

interface CommandItemLists {
  lists?: { items?: TextVersion[] }[]
}

const listVersions = async (request: APIRequestContext, filePath: string): Promise<TextVersion[]> => {
  const answer = (await callRPC(request, 'CommandService/ListCommandItems', {
    path: filePath,
    commands: [{ command: DELETE_COMMAND }],
  })) as CommandItemLists
  return (answer.lists ?? []).flatMap((list) => list.items ?? [])
}

export const listVersionIds = async (request: APIRequestContext, filePath: string): Promise<string[]> =>
  (await listVersions(request, filePath)).map((one) => one.id)

export const deleteVersion = async (request: APIRequestContext, filePath: string, id: string): Promise<void> => {
  await callRPC(request, 'CommandService/RunPluginCommand', { command: DELETE_COMMAND, path: filePath, itemId: id })
}

export const hasMadeText = async (request: APIRequestContext, filePath: string): Promise<boolean> =>
  (await listVersions(request, filePath)).some((one) => !one.title.startsWith(TEXT_LAYER))

export const hasVersions = async (request: APIRequestContext, filePath: string): Promise<boolean> =>
  (await listVersions(request, filePath)).length > 0
