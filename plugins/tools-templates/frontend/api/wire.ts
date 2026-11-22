/**
 * Wire adapter for the plugin's templates service.
 */
import { createApi, type ApiError, type Result } from '@numen/plugin-sdk'
import {
  SERVICE,
  type CreateNoteRequest,
  type CreateNoteResponse,
  type ListFieldsRequest,
  type ListFieldsResponse,
} from './templates'

/** What the command asks of the backend. */
export interface TemplatesApi {
  /** Lists what a template asks of the person before a note is made from it. */
  listFields(request: ListFieldsRequest): Promise<Result<ListFieldsResponse, ApiError>>
  /** Makes a note from a template. */
  createNote(request: CreateNoteRequest): Promise<Result<CreateNoteResponse, ApiError>>
}

const api = createApi(SERVICE)

export const templatesApi: TemplatesApi = {
  listFields: (request) => api.call('ListFields', request),
  createNote: (request) => api.call('CreateNote', request),
}
