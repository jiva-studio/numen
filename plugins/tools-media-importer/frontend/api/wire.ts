/**
 * Wire adapter for ArticleService, RecordingService and the window's tasks.
 */
import { createClient } from '@connectrpc/connect'
import { ArticleErrorCode, ArticleService, RecordingService, WindowService } from '@numen/protocol'
import { retryWhileBusy, transport } from '@numen/wire'
import type { Links } from '../types'
import { WORDS as words } from '../words'
import { createWorkWatcher } from './work'

const client = {
  articles: createClient(ArticleService, transport),
  recordings: createClient(RecordingService, transport),
  window: createClient(WindowService, transport),
}

/** The window the plugin's tabs are drawn in. */
const WINDOW = 'editor'

/** What a person reads for each refusal. */
const REFUSALS: Readonly<Record<ArticleErrorCode, string>> = {
  [ArticleErrorCode.UNSPECIFIED]: words.failed,
  [ArticleErrorCode.NOT_AN_ADDRESS]: words.notAnAddress,
  [ArticleErrorCode.OCCUPIED]: words.occupied,
  [ArticleErrorCode.NOT_A_LINK]: words.notALink,
  [ArticleErrorCode.NO_VAULT]: words.noVault,
}

/** What a person reads for a refusal of the schema. */
export const describeRefusal = (code: ArticleErrorCode): string => REFUSALS[code] ?? words.failed

const work = createWorkWatcher(async function* (signal) {
  for await (const update of client.window.watchTasks({ window: WINDOW }, { signal })) {
    yield update.tasks
  }
})

export const links: Links = {
  getLink: async (path) => {
    const answer = await retryWhileBusy(() => client.articles.getLink({ path }))
    if (answer.error !== ArticleErrorCode.UNSPECIFIED) {
      return { ok: false, error: describeRefusal(answer.error) }
    }
    return {
      ok: true,
      value: { url: answer.url, isCopyable: answer.isCopyable, copySize: Number(answer.copySize) },
    }
  },
  readArticle: async (path) =>
    (await retryWhileBusy(() => client.articles.readArticle({ path }))).text,
  getSource: async (path) => {
    const answer = await retryWhileBusy(() => client.recordings.getRecording({ path }))
    return { mediaUrl: answer.mediaUrl, mediaType: answer.mediaType }
  },
  createLink: async (folder, url) => {
    const answer = await client.articles.createLink({ folder, url })
    if (answer.error !== ArticleErrorCode.UNSPECIFIED) {
      return { ok: false, error: describeRefusal(answer.error) }
    }
    return { ok: true, value: { path: answer.path } }
  },
  watchWork: (path, onChange) => work.watch(path, onChange),
}
