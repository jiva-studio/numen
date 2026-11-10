/**
 * The client of the JSON methods a plugin's backend serves.
 *
 * A method is a POST of JSON to `/<service>/<method>`; the window forwards it to the plugin that
 * declares the service in its manifest. A stream is a method answered as Server-Sent Events.
 */
import { getOrigin } from './transport'
import { asFailure, asValue } from './result'
import type { Result } from './result'
import { parseEvents } from './stream'

/** Why a method failed: a code such as `not_found` and the text the backend gave. */
export interface ApiError {
  readonly code: string
  readonly message: string
}

/** The code of a failure that never reached the method: the network or the host answered. */
export const UNAVAILABLE = 'unavailable'

interface ApiOptions {
  readonly origin?: string
  readonly fetch?: typeof fetch
}

interface ApiCallOptions {
  readonly signal?: AbortSignal
}

export interface Api {
  /** Calls a method and returns its response or its failure. */
  call<Req, Res>(
    method: string,
    request: Req,
    options?: ApiCallOptions,
  ): Promise<Result<Res, ApiError>>
  /**
   * Calls a stream method and yields each response as it arrives. The generator
   * ends with the stream, or returns the failure that ended it.
   */
  stream<Req, Res>(
    method: string,
    request: Req,
    options?: ApiCallOptions,
  ): AsyncGenerator<Res, ApiError | null>
}

const readFailure = async (response: Response): Promise<ApiError> => {
  try {
    const body = (await response.json()) as Partial<ApiError>
    if (typeof body.code === 'string' && typeof body.message === 'string') {
      return { code: body.code, message: body.message }
    }
  } catch {
    return { code: UNAVAILABLE, message: `HTTP ${response.status}` }
  }
  return { code: UNAVAILABLE, message: `HTTP ${response.status}` }
}

const createRequest = <Req>(
  request: Req | undefined,
  signal: AbortSignal | undefined,
  accept: string,
): RequestInit => ({
  method: 'POST',
  headers: { 'Content-Type': 'application/json', Accept: accept },
  body: JSON.stringify(request ?? {}),
  signal: signal ?? null,
})

async function* readEvents(
  body: NonNullable<Response['body']>,
): AsyncGenerator<{ name: string; data: string }> {
  const reader = body.pipeThrough(new TextDecoderStream()).getReader()
  let pending = ''
  try {
    for (;;) {
      const chunk = await reader.read()
      if (chunk.done) return
      const parsed = parseEvents(pending + chunk.value)
      pending = parsed.rest
      yield* parsed.events
    }
  } finally {
    await reader.cancel().catch(() => undefined)
  }
}

/** Creates the client of the methods the service serves. */
export const createApi = (service: string, options: ApiOptions = {}): Api => {
  const getUrl = (method: string): string => `${options.origin ?? getOrigin()}/${service}/${method}`
  const send = (url: string, init: RequestInit): Promise<Response> =>
    (options.fetch ?? fetch)(url, init)

  return {
    async call<Req, Res>(method: string, request: Req, callOptions: ApiCallOptions = {}) {
      let response: Response
      try {
        response = await send(
          getUrl(method),
          createRequest(request, callOptions.signal, 'application/json'),
        )
      } catch (error) {
        return asFailure({
          code: UNAVAILABLE,
          message: error instanceof Error ? error.message : String(error),
        })
      }
      if (!response.ok) return asFailure(await readFailure(response))
      return asValue((await response.json()) as Res)
    },

    async *stream<Req, Res>(method: string, request: Req, callOptions: ApiCallOptions = {}) {
      let response: Response
      try {
        response = await send(
          getUrl(method),
          createRequest(request, callOptions.signal, 'text/event-stream'),
        )
      } catch (error) {
        return {
          code: UNAVAILABLE,
          message: error instanceof Error ? error.message : String(error),
        }
      }
      if (!response.ok || !response.body) return await readFailure(response)

      for await (const event of readEvents(response.body)) {
        if (event.name === 'end') return null
        if (event.name === 'error') return JSON.parse(event.data) as ApiError
        yield JSON.parse(event.data) as Res
      }
      return null
    },
  }
}
