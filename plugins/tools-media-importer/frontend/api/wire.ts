/**
 * Wire adapter for the plugin's LinkService, the window's MediaService and the plugin's tasks.
 */
import { createClient } from '@connectrpc/connect'
import { MediaService } from '@numen/protocol'
import type { TaskHost } from '@numen/plugin-sdk'
import { retryWhileBusy, transport } from '@numen/wire'
import { LinkErrorCode, LinkService } from '../gen/numen/links/v1/link_pb.js'
import type { Links, Outcome } from '../types'
import { WORDS as words } from '../words'
import { createWorkWatcher } from './work'

/** The id the manifest declares the plugin under. */
export const PLUGIN_ID = 'md.numen.tools.media.importer'

const client = {
  links: createClient(LinkService, transport),
  media: createClient(MediaService, transport),
}

let taskHost: TaskHost | undefined

/** Sets the task host the plugin was activated with. */
export const setTaskHost = (host: TaskHost | undefined) => {
  taskHost = host
}

/** What a person reads for each refusal. */
const REFUSALS: Readonly<Record<LinkErrorCode, string>> = {
  [LinkErrorCode.UNSPECIFIED]: words.failed,
  [LinkErrorCode.NOT_A_LINK]: words.notALink,
  [LinkErrorCode.NO_VAULT]: words.noVault,
}

/** What a person reads for a refusal of the schema. */
export const describeRefusal = (code: LinkErrorCode): string => REFUSALS[code] ?? words.failed

/** The outcome of a call that returns nothing but a refusal. */
const readOutcome = (answer: { readonly error: LinkErrorCode }): Outcome<null> =>
  answer.error === LinkErrorCode.UNSPECIFIED
    ? { ok: true, value: null }
    : { ok: false, error: describeRefusal(answer.error) }

const work = createWorkWatcher((onTasks) => taskHost?.watch(onTasks) ?? { dispose: () => {} })

export const links: Links = {
  getLink: async (path) => {
    const answer = await retryWhileBusy(() => client.links.getLink({ path }))
    if (answer.error !== LinkErrorCode.UNSPECIFIED) {
      return { ok: false, error: describeRefusal(answer.error) }
    }
    return {
      ok: true,
      value: {
        url: answer.url,
        isCopyable: answer.isCopyable,
        copySize: Number(answer.copySize),
        isTextDeleted: answer.isTextDeleted,
        embedPage: answer.embedPage,
      },
    }
  },
  readArticle: async (path) =>
    (await retryWhileBusy(() => client.links.readArticle({ path }))).text,
  getSource: async (path) => {
    const answer = await retryWhileBusy(() => client.media.getMediaSource({ path }))
    return { mediaUrl: answer.src, mediaType: answer.mediaType }
  },
  getPageSource: async (page) =>
    (await retryWhileBusy(() => client.media.getPageSource({ pluginId: PLUGIN_ID, path: page })))
      .src,
  fetchText: async (path, isUserStarted) =>
    readOutcome(await client.links.fetchText({ path, isUserStarted })),
  copyVideo: async (path) => readOutcome(await client.links.copyVideo({ path })),
  deleteCopy: async (path) => readOutcome(await client.links.deleteCopy({ path })),
  watchWork: (path, onChange) => work.watch(path, onChange),
}

/** Makes a link note of each web address in the folder; answers the notes made. */
export const createLinks = async (
  folder: string,
  urls: readonly string[],
): Promise<Outcome<readonly string[]>> => {
  const answer = await client.links.createLinks({ folder, urls: [...urls] })
  return answer.error === LinkErrorCode.UNSPECIFIED
    ? { ok: true, value: answer.paths }
    : { ok: false, error: describeRefusal(answer.error) }
}
