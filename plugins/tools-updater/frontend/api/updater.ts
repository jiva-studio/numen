/**
 * The JSON messages of the updater service the plugin's backend serves. Keys and meaning match the
 * Go messages in backend/api; testdata/api holds the fixtures both sides are tested against.
 */

import type { Platform } from '../types'

/** The service name the manifest declares. */
export const SERVICE = 'md.numen.tools.updater'

/** What the last check found. */
type State = 'unchecked' | 'current' | 'available' | 'unreachable'

/** The running build, as the backend compares it with the newest one. */
export interface App {
  readonly version: string
  readonly build: number
  readonly platform: Platform
}

interface CheckRequest {
  readonly app: App
}

type GetStatusRequest = Record<string, never>

/** The file of a release for the running platform. */
interface ReleaseFile {
  readonly name: string
  readonly arch: string
  readonly kind: string
  readonly url: string
  readonly size: number
  readonly sha256: string
}

/** A build newer than the running one. */
export interface Update {
  readonly version: string
  readonly build: number
  readonly channel: string
  readonly releasedAt: string
  readonly notesUrl: string
  readonly downloadUrl: string
  readonly file?: ReleaseFile
}

/** The result of the last check. */
export interface Status {
  readonly state: State
  readonly update?: Update
}

/** The request and response of each method by name. */
export interface Methods {
  Check: { request: CheckRequest; response: Status }
  GetStatus: { request: GetStatusRequest; response: Status }
}
