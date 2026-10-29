/**
 * The client of the JSON methods a plugin's backend serves.
 *
 * A method is a POST of JSON to `/<service>/<method>`; the window forwards it to the plugin that
 * declares the service in its manifest. A stream is a method answered as Server-Sent Events.
 */
import { getOrigin } from './transport'
import { asFailure, asValue } from './result'
import type { Result } from './result'

/** Why a method failed: a code such as `not_found` and the text the backend gave. */
export interface ApiError {
  readonly code: string
  readonly message: string
}

/** The code of a failure that never reached the method: the network or the host answered. */
export const UNAVAILABLE = 'unavailable'

export interface ApiOptions {
  /** The origin the methods are asked at; the origin of the page where it is empty. */
  readonly origin?: string
  /** The function that makes the requests, for a window without one or a test. */
  readonly fetch?: typeof fetch
}

export interface ApiCallOptions {
  readonly signal?: AbortSignal
}

export interface Api {
  /** Asks a method and returns its response or its failure. */
  call<Req, Res>(
    method: string,
    request: Req,
    options?: ApiCallOptions,
  ): Promise<Result<Res, ApiError>>
  /**
   * Asks a stream method and yields each response as it arrives. The generator ends with the
   * stream, or returns the failure that ended it.
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
    // the body is not a failure of the method
  }
  return { code: UNAVAILABLE, message: `HTTP ${response.status}` }
}

const createRequest = (
  request: unknown,
  signal: AbortSignal | undefined,
  accept: string,
): RequestInit => ({
  method: 'POST',
  headers: { 'Content-Type': 'application/json', Accept: accept },
  body: JSON.stringify(request ?? {}),
  signal: signal ?? null,
})

/** Splits the text of an event stream into its events as they complete. */
export const parseEvents = (
  text: string,
): { events: { name: string; data: string }[]; rest: string } => {
  const events: { name: string; data: string }[] = []
  const blocks = text.split('\n\n')
  const rest = blocks.pop() ?? ''
  for (const block of blocks) {
    let name = 'message'
    const data: string[] = []
    for (const line of block.split('\n')) {
      if (line.startsWith('event: ')) name = line.slice(7)
      else if (line.startsWith('data: ')) data.push(line.slice(6))
    }
    events.push({ name, data: data.join('\n') })
  }
  return { events, rest }
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

      const reader = response.body.pipeThrough(new TextDecoderStream()).getReader()
      let pending = ''
      try {
        for (;;) {
          const chunk = await reader.read()
          if (chunk.done) return null
          const parsed = parseEvents(pending + chunk.value)
          pending = parsed.rest
          for (const event of parsed.events) {
            if (event.name === 'end') return null
            if (event.name === 'error') return JSON.parse(event.data) as ApiError
            yield JSON.parse(event.data) as Res
          }
        }
      } finally {
        await reader.cancel().catch(() => undefined)
      }
    },
  }
}
