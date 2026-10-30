/**
 * Wire adapter for the plugin's importer service, the window's MediaService and the plugin's tasks.
 */
import {
  createApi,
  getMediaSource,
  getPageSource,
  type ApiError,
  type Result,
  type TaskHost,
  UNAVAILABLE,
} from '@numen/plugin-sdk'
import type { Links, Outcome } from '../types'
import { WORDS as words } from '../words'
import { ErrorCode, SERVICE, type Methods } from './importer'
import { createWorkWatcher } from './work'

/** The id the manifest declares the plugin under. */
export const PLUGIN_ID = 'md.numen.tools.media.importer'

const BUSY_ATTEMPTS = 5
const BUSY_DELAY_MS = 1000

const api = createApi(SERVICE)

let taskHost: TaskHost | undefined

/** Sets the task host the plugin was activated with. */
export const setTaskHost = (host: TaskHost | undefined) => {
  taskHost = host
}

/** The message displayed to the user for a method failure. */
export const describeError = (error: ApiError): string => {
  switch (error.code) {
    case ErrorCode.notFound:
      return words.notALink
    case ErrorCode.failedPrecondition:
      return words.noVault
    default:
      return words.failed
  }
}

/** Calls a method of the importer service. */
const call = <M extends keyof Methods>(
  method: M,
  request: Methods[M]['request'],
): Promise<Result<Methods[M]['response'], ApiError>> => api.call(method, request)

/** Calls a method again after a wait while the application answers unavailable. */
const callWhileBusy = async <M extends keyof Methods>(
  method: M,
  request: Methods[M]['request'],
): Promise<Result<Methods[M]['response'], ApiError>> => {
  for (let attempt = 1; ; attempt++) {
    const result = await call(method, request)
    if (result.ok || result.error.code !== UNAVAILABLE || attempt >= BUSY_ATTEMPTS) return result
    await new Promise((resolve) => setTimeout(resolve, BUSY_DELAY_MS))
  }
}

/** The outcome of a call whose response holds no value. */
const readOutcome = (result: Result<unknown, ApiError>): Outcome<null> =>
  result.ok ? { ok: true, value: null } : { ok: false, error: describeError(result.error) }

// Without a task host there is nothing to unsubscribe.
const work = createWorkWatcher(
  (onTasks) => taskHost?.watch(onTasks) ?? { dispose: () => undefined },
)

/** Notifies open tabs on the named paths to reload their link. */
export const refreshLinks = (paths: readonly string[]): void => {
  work.refresh(paths)
}

export const links: Links = {
  getLink: async (path) => {
    const result = await callWhileBusy('GetLink', { path })
    if (!result.ok) return { ok: false, error: describeError(result.error) }
    return { ok: true, value: result.value }
  },
  readArticle: async (path) => {
    const result = await callWhileBusy('ReadArticle', { path })
    if (!result.ok) throw new Error(result.error.message)
    return result.value.text
  },
  getSource: (path) => getMediaSource(path),
  getPageSource: (page) => getPageSource(PLUGIN_ID, page),
  fetchText: async (path, isUserStarted) =>
    readOutcome(await call('FetchText', { path, isUserStarted })),
  copyVideo: async (path) => readOutcome(await call('CopyVideo', { path })),
  deleteCopy: async (path) => readOutcome(await call('DeleteCopy', { path })),
  watchWork: (path, onChange) => work.watch(path, onChange),
}

/** Creates a link note of each web address in the folder; returns the paths of the notes. */
export const createLinks = async (
  folder: string,
  urls: readonly string[],
): Promise<Outcome<readonly string[]>> => {
  const result = await call('CreateLinks', { folder, urls })
  return result.ok
    ? { ok: true, value: result.value.paths }
    : { ok: false, error: describeError(result.error) }
}
