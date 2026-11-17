/**
 * Wire adapter for the plugin's importer service.
 */
import { createApi, type ApiError, type Result, UNAVAILABLE } from '@numen/plugin-sdk'
import type { DroppedLinks, Links, Outcome } from '../types'
import { SERVICE, type Methods } from './importer'
import { describeError } from './failure'

const BUSY_ATTEMPTS = 5
const BUSY_DELAY_MS = 1000

const api = createApi(SERVICE)

const call = <M extends keyof Methods>(
  method: M,
  request: Methods[M]['request'],
): Promise<Result<Methods[M]['response'], ApiError>> =>
  api.call<Methods[M]['request'], Methods[M]['response']>(method, request)

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

const readOutcome = <Response>(result: Result<Response, ApiError>): Outcome<null> =>
  result.ok ? { ok: true, value: null } : { ok: false, error: describeError(result.error) }

export const links: Links = {
  getLink: async (path) => {
    const result = await callWhileBusy('GetLink', { path })
    if (!result.ok) return { ok: false, error: describeError(result.error) }
    return { ok: true, value: result.value }
  },
  copyMedia: async (path, kind) => readOutcome(await call('CopyMedia', { path, kind })),
}

/** Creates a link note of each web address in the folder; returns the paths of the notes and the addresses that failed. */
export const createLinks = async (
  folder: string,
  urls: readonly string[],
): Promise<Outcome<DroppedLinks>> => {
  const result = await call('CreateLinks', { folder, urls })
  return result.ok
    ? { ok: true, value: { paths: result.value.paths, failed: result.value.failed } }
    : { ok: false, error: describeError(result.error) }
}
