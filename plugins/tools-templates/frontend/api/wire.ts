/**
 * Wire adapter for the plugin's templates service.
 */
import { createApi, type ApiError, type Result } from '@numen/plugin-sdk'
import { SERVICE, type CreateNoteRequest, type CreateNoteResponse } from './templates'

/** What the command asks of the backend. */
export interface TemplatesApi {
  /** Makes a note from a template. */
  createNote(request: CreateNoteRequest): Promise<Result<CreateNoteResponse, ApiError>>
}

const api = createApi(SERVICE)

export const templatesApi: TemplatesApi = {
  createNote: (request) => api.call('CreateNote', request),
}
