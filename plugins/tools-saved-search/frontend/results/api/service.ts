/**
 * The calls of the saved search tab to the backend, over the JSON service of the plugin.
 */
import { createApi, type Api, type ApiError } from '@numen/plugin-sdk'
import type {
  GetResultsRequest,
  GetResultsResponse,
  ListSearchFunctionsRequest,
  ListSearchFunctionsResponse,
  ResultsChange,
  UpdateSearchRequest,
  UpdateSearchResponse,
  WatchResultsRequest,
} from './types'
import { SERVICE } from './serviceName'
import { ServiceFailure } from './serviceFailure'

/** The calls of the tab to the backend. */
export interface SavedSearchService {
  getResults(request: GetResultsRequest, signal: AbortSignal): Promise<GetResultsResponse>
  watchResults(
    request: WatchResultsRequest,
    signal: AbortSignal,
  ): AsyncGenerator<ResultsChange, void>
  updateSearch(request: UpdateSearchRequest, signal: AbortSignal): Promise<UpdateSearchResponse>
  listSearchFunctions(
    request: ListSearchFunctionsRequest,
    signal: AbortSignal,
  ): Promise<ListSearchFunctionsResponse>
}

const readEnd = (failure: ApiError | null, signal: AbortSignal): void => {
  if (failure && !signal.aborted) throw new ServiceFailure(failure.message)
}

/** The service over the API; a failed call throws ServiceFailure with the message of the failure. */
export function createSavedSearchService(api: Api = createApi(SERVICE)): SavedSearchService {
  const call = async <Req, Res>(
    method: string,
    request: Req,
    signal: AbortSignal,
  ): Promise<Res> => {
    const answer = await api.call<Req, Res>(method, request, { signal })
    if (!answer.ok) throw new ServiceFailure(answer.error.message)
    return answer.value
  }

  return {
    getResults: (request, signal) =>
      call<GetResultsRequest, GetResultsResponse>('GetResults', request, signal),

    async *watchResults(request, signal) {
      const changes = api.stream<WatchResultsRequest, ResultsChange>('WatchResults', request, {
        signal,
      })
      try {
        for (;;) {
          const next = await changes.next()
          if (next.done) return readEnd(next.value, signal)
          yield next.value
        }
      } catch (error) {
        if (signal.aborted) return
        throw error
      }
    },

    updateSearch: (request, signal) =>
      call<UpdateSearchRequest, UpdateSearchResponse>('UpdateSearch', request, signal),

    listSearchFunctions: (request, signal) =>
      call<ListSearchFunctionsRequest, ListSearchFunctionsResponse>(
        'ListSearchFunctions',
        request,
        signal,
      ),
  }
}
