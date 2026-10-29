/**
 * The client of the `numen.flashcards` service: one function for each method of the backend. A
 * function answers with the response of its method and throws an `ApiFailure` when the method fails.
 * A stream function yields each response as it arrives and throws when the stream ends in failure.
 */
import { createApi } from '@numen/plugin-sdk'
import { ApiFailure, CANCELED } from './failure'
import type {
  AnswerCardRequest,
  AnswerCardResponse,
  ComputeCurveRequest,
  ComputeCurveResponse,
  CreateCardModelRequest,
  CreateCardModelResponse,
  CreateDeckRequest,
  CreateDeckResponse,
  CreatePresetRequest,
  CreatePresetResponse,
  GetDeckNeighbourhoodRequest,
  GetDeckNeighbourhoodResponse,
  GetDeckPresetRequest,
  GetDeckPresetResponse,
  GetVaultDeckPresetRequest,
  GetVaultDeckPresetResponse,
  ListCardModelsRequest,
  ListCardModelsResponse,
  ListPresetsRequest,
  ListPresetsResponse,
  ListReviewDaysRequest,
  ListReviewDaysResponse,
  ReadCardModelRequest,
  ReadCardModelResponse,
  ReadDeckRequest,
  ReadDeckResponse,
  ReadPresetRequest,
  ReadPresetResponse,
  RenameCardModelFieldRequest,
  RenameCardModelFieldResponse,
  ScheduleDeckRequest,
  ScheduleDeckResponse,
  StartSessionRequest,
  StartSessionResponse,
  TakeBackAnswerRequest,
  TakeBackAnswerResponse,
  WatchCardsDueRequest,
  WatchCardsDueResponse,
  WatchReloadsRequest,
  WatchReloadsResponse,
  WriteCardModelRequest,
  WriteCardModelResponse,
  WriteDeckRequest,
  WriteDeckResponse,
  WritePresetRequest,
  WritePresetResponse,
} from './types'

/** The name the manifest declares for the service. */
export const SERVICE = 'numen.flashcards'

/** Options of one call. */
export interface CallOptions {
  readonly signal?: AbortSignal
}

const api = createApi(SERVICE)

const call = async <Req, Res>(
  method: string,
  request: Req,
  options?: CallOptions,
): Promise<Res> => {
  const answer = await api.call<Req, Res>(method, request, options)
  if (!answer.ok) throw new ApiFailure(options?.signal?.aborted ? CANCELED : answer.error)
  return answer.value
}

async function* watch<Req, Res>(
  method: string,
  request: Req,
  options?: CallOptions,
): AsyncGenerator<Res, void> {
  const responses = api.stream<Req, Res>(method, request, options)
  for (;;) {
    const next = await responses.next()
    if (next.done) {
      if (next.value) throw new ApiFailure(options?.signal?.aborted ? CANCELED : next.value)
      return
    }
    yield next.value
  }
}

/** The decks and the card models of a vault. */
export const cardsService = {
  listCardModels: (request: ListCardModelsRequest, options?: CallOptions) =>
    call<ListCardModelsRequest, ListCardModelsResponse>('ListCardModels', request, options),
  createCardModel: (request: CreateCardModelRequest, options?: CallOptions) =>
    call<CreateCardModelRequest, CreateCardModelResponse>('CreateCardModel', request, options),
  readCardModel: (request: ReadCardModelRequest, options?: CallOptions) =>
    call<ReadCardModelRequest, ReadCardModelResponse>('ReadCardModel', request, options),
  writeCardModel: (request: WriteCardModelRequest, options?: CallOptions) =>
    call<WriteCardModelRequest, WriteCardModelResponse>('WriteCardModel', request, options),
  renameCardModelField: (request: RenameCardModelFieldRequest, options?: CallOptions) =>
    call<RenameCardModelFieldRequest, RenameCardModelFieldResponse>(
      'RenameCardModelField',
      request,
      options,
    ),
  createDeck: (request: CreateDeckRequest, options?: CallOptions) =>
    call<CreateDeckRequest, CreateDeckResponse>('CreateDeck', request, options),
  readDeck: (request: ReadDeckRequest, options?: CallOptions) =>
    call<ReadDeckRequest, ReadDeckResponse>('ReadDeck', request, options),
  writeDeck: (request: WriteDeckRequest, options?: CallOptions) =>
    call<WriteDeckRequest, WriteDeckResponse>('WriteDeck', request, options),
}

/** The presets that decks are scheduled by. */
export const presetsService = {
  getDeckPreset: (request: GetDeckPresetRequest, options?: CallOptions) =>
    call<GetDeckPresetRequest, GetDeckPresetResponse>('GetDeckPreset', request, options),
  listPresets: (request: ListPresetsRequest, options?: CallOptions) =>
    call<ListPresetsRequest, ListPresetsResponse>('ListPresets', request, options),
  createPreset: (request: CreatePresetRequest, options?: CallOptions) =>
    call<CreatePresetRequest, CreatePresetResponse>('CreatePreset', request, options),
  scheduleDeck: (request: ScheduleDeckRequest, options?: CallOptions) =>
    call<ScheduleDeckRequest, ScheduleDeckResponse>('ScheduleDeck', request, options),
  readPreset: (request: ReadPresetRequest, options?: CallOptions) =>
    call<ReadPresetRequest, ReadPresetResponse>('ReadPreset', request, options),
  writePreset: (request: WritePresetRequest, options?: CallOptions) =>
    call<WritePresetRequest, WritePresetResponse>('WritePreset', request, options),
  computeCurve: (request: ComputeCurveRequest, options?: CallOptions) =>
    call<ComputeCurveRequest, ComputeCurveResponse>('ComputeCurve', request, options),
}

/** The counts, the sessions and the answers of the review. Every call names the vault it is about. */
export const reviewService = {
  watchCardsDue: (request: WatchCardsDueRequest, options?: CallOptions) =>
    watch<WatchCardsDueRequest, WatchCardsDueResponse>('WatchCardsDue', request, options),
  startSession: (request: StartSessionRequest, options?: CallOptions) =>
    call<StartSessionRequest, StartSessionResponse>('StartSession', request, options),
  answerCard: (request: AnswerCardRequest, options?: CallOptions) =>
    call<AnswerCardRequest, AnswerCardResponse>('AnswerCard', request, options),
  takeBackAnswer: (request: TakeBackAnswerRequest, options?: CallOptions) =>
    call<TakeBackAnswerRequest, TakeBackAnswerResponse>('TakeBackAnswer', request, options),
  watchReloads: (request: WatchReloadsRequest, options?: CallOptions) =>
    watch<WatchReloadsRequest, WatchReloadsResponse>('WatchReloads', request, options),
  listReviewDays: (request: ListReviewDaysRequest, options?: CallOptions) =>
    call<ListReviewDaysRequest, ListReviewDaysResponse>('ListReviewDays', request, options),
  getDeckNeighbourhood: (request: GetDeckNeighbourhoodRequest, options?: CallOptions) =>
    call<GetDeckNeighbourhoodRequest, GetDeckNeighbourhoodResponse>(
      'GetDeckNeighbourhood',
      request,
      options,
    ),
  getVaultDeckPreset: (request: GetVaultDeckPresetRequest, options?: CallOptions) =>
    call<GetVaultDeckPresetRequest, GetVaultDeckPresetResponse>(
      'GetVaultDeckPreset',
      request,
      options,
    ),
}
