/**
 * Wire adapter for the plugin's templates service.
 */
import type { ApiError, Result } from '@numen/plugin-sdk'
import type { CreateNoteRequest, CreateNoteResponse } from './templates'

/** What the command asks of the backend. */
export interface TemplatesApi {
  /** Makes a note from a template. */
  createNote(request: CreateNoteRequest): Promise<Result<CreateNoteResponse, ApiError>>
}

export const templatesApi: TemplatesApi = {
  createNote: () => Promise.reject(new Error('not implemented')),
}
