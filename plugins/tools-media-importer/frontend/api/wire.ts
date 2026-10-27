/**
 * Wire adapter for ArticleService, RecordingService and the plugin's tasks.
 */
import { createClient } from '@connectrpc/connect'
import { ArticleErrorCode, ArticleService, RecordingService } from '@numen/protocol'
import type { TaskHost } from '@numen/plugin-sdk'
import { retryWhileBusy, transport } from '@numen/wire'
import type { Links } from '../types'
import { WORDS as words } from '../words'
import { createWorkWatcher } from './work'

const client = {
  articles: createClient(ArticleService, transport),
  recordings: createClient(RecordingService, transport),
}

let taskHost: TaskHost | undefined

/** Sets the task host the plugin was activated with. */
export const setTaskHost = (host: TaskHost | undefined) => {
  taskHost = host
}

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

const work = createWorkWatcher((onTasks) => taskHost?.watch(onTasks) ?? { dispose: () => {} })

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
