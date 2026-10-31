/**
 * Wire adapter for the plugin's relevance service: each call of the panel's client is one JSON method.
 */
import { createApi, type Api, type ApiError } from '@numen/plugin-sdk'
import { SERVICE, type FloorClient, type Methods } from './relevance'

/** The error the panel shows for a failed method: the code and the text the backend gave. */
function createFailure(error: ApiError): Error {
  return new Error(`[${error.code}] ${error.message}`)
}

/** The client of the panel over the methods of the relevance service. */
export function createFloorClient(api: Api = createApi(SERVICE)): FloorClient {
  async function call<M extends keyof Methods>(
    method: M,
    request: Methods[M]['request'],
  ): Promise<Methods[M]['response']> {
    const result = await api.call<Methods[M]['request'], Methods[M]['response']>(method, request)
    if (!result.ok) throw createFailure(result.error)
    return result.value
  }

  return {
    getFloor: () => call('GetFloor', {}),
    calibrateFloor: () => call('CalibrateFloor', {}),
    setFloor: (request) => call('SetFloor', request),
    probeFloor: (request) => call('ProbeFloor', request),
  }
}
