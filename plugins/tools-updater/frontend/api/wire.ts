/**
 * Wire adapter for the plugin's updater service.
 */
import { createApi, type ApiError, type Result } from '@numen/plugin-sdk'
import { SERVICE, type App, type Status } from './updater'

const api = createApi(SERVICE)

/** What the model asks of the backend. */
export interface UpdateApi {
  /** Asks the backend to compare the running build with the newest one. */
  check(app: App): Promise<Result<Status, ApiError>>
}

export const updater: UpdateApi = {
  check: (app) => api.call<{ app: App }, Status>('Check', { app }),
}
