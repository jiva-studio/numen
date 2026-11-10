/** What a client has left when it reports a flush. */
export type FlushOutcome = 'nothing' | 'written' | 'unsaved'

/** A request of the application that the window write what is pending. */
export interface QuitRequest {
  readonly token: string
  readonly flush: boolean
}
