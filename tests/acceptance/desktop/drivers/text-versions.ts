/** The text versions of a file (the text layer, a recognition, a proofreading), as the host lists and deletes them. */
import type { APIRequestContext } from 'playwright/test'
import { callRPC } from '../../../e2e/desktop/tests/support/vault.js'

/** The title of the version a file has before any plugin makes text of it. */
const TEXT_LAYER = 'Text layer'

export interface TextVersion {
  id: string
  title: string
}

interface VersionItems {
  items?: TextVersion[]
}

export const listVersions = async (request: APIRequestContext, filePath: string): Promise<TextVersion[]> => {
  const answer = (await callRPC(request, 'TextVersionService/ListVersionItems', { path: filePath })) as VersionItems
  return answer.items ?? []
}

export const listVersionIds = async (request: APIRequestContext, filePath: string): Promise<string[]> =>
  (await listVersions(request, filePath)).map((one) => one.id)

export const deleteVersion = async (request: APIRequestContext, filePath: string, id: string): Promise<void> => {
  await callRPC(request, 'TextVersionService/DeleteVersion', { path: filePath, versionId: id })
}

export const hasMadeText = async (request: APIRequestContext, filePath: string): Promise<boolean> =>
  (await listVersions(request, filePath)).some((one) => !one.title.startsWith(TEXT_LAYER))

export const hasVersions = async (request: APIRequestContext, filePath: string): Promise<boolean> =>
  (await listVersions(request, filePath)).length > 0

export const hasTextLayerVersion = async (request: APIRequestContext, filePath: string): Promise<boolean> =>
  (await listVersions(request, filePath)).some((one) => one.title.startsWith(TEXT_LAYER))
